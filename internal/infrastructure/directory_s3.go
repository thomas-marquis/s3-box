package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

const (
	nbWorkers = 5
)

type s3Session struct {
	connection *connection_deck.Connection
	client     *s3.S3
	session    *session.Session
}

type S3DirectoryRepository struct {
	sync.Mutex
	connectionRepository *FyneConnectionsRepository
	logger               *log.Logger
	cache                map[connection_deck.ConnectionID]*s3Session
}

var _ directory.Repository = &S3DirectoryRepository{}

func NewS3DirectoryRepository(
	connectionsRepository *FyneConnectionsRepository,
	publisher *directory.EventPublisher,
	errorStream chan error,
	terminate chan struct{},
) (*S3DirectoryRepository, error) {
	r := &S3DirectoryRepository{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		cache:                make(map[connection_deck.ConnectionID]*s3Session),
	}

	events := make(chan directory.Event)
	publisher.Subscribe(events)

	go func() {
		<-terminate
		publisher.Unsubscribe(events)
		close(events)
	}()

	for i := 0; i < nbWorkers; i++ {
		go r.listen(events, errorStream)
	}

	return r, nil
}

func (r *S3DirectoryRepository) GetByPath(ctx context.Context, connID connection_deck.ConnectionID, path directory.Path) (*directory.Directory, error) {
	searchKey := mapPathToSearchKey(path)

	s, err := r.getSession(ctx, connID)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	inputs := &s3.ListObjectsInput{
		Bucket:    aws.String(s.connection.Bucket()),
		Prefix:    aws.String(searchKey),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int64(1000),
	}

	dir, err := directory.New(connID, path.DirectoryName(), path.ParentPath())
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	pageHandler := func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			key := *obj.Key
			if key == searchKey {
				continue
			}
			if _, err := dir.NewFile(mapKeyToObjectName(key),
				directory.WithFileSize(int(*obj.Size)),
				directory.WithFileLastModified(*obj.LastModified),
			); err != nil {
				return false
			}
		}

		for _, obj := range page.CommonPrefixes {
			if *obj.Prefix == searchKey {
				continue
			}
			s3Prefix := *obj.Prefix
			isDir := strings.HasSuffix(s3Prefix, "/")
			if isDir {
				if _, err := dir.NewSubDirectory(mapKeyToObjectName(s3Prefix)); err != nil {
					return false
				}
			}
		}
		return !lastPage
	}

	if err := s.client.ListObjectsPagesWithContext(ctx, inputs, pageHandler); err != nil {
		return nil, r.manageAwsSdkError(err, searchKey, s)
	}

	return dir, nil
}

func (r *S3DirectoryRepository) GetFileContent(ctx context.Context, connId connection_deck.ConnectionID, file *directory.File) (*directory.Content, error) {
	s, err := r.getSession(ctx, connId)
	if err != nil {
		return nil, fmt.Errorf("GetFileContent: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.connection.Bucket()),
		Key:    aws.String(mapFileToKey(file)),
	}

	result, err := s.client.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, r.manageAwsSdkError(err, file.FullPath(), s)
	}

	defer result.Body.Close()

	buff := new(bytes.Buffer)
	if _, err = buff.ReadFrom(result.Body); err != nil {
		return nil, fmt.Errorf("fail reading the body content: %w", err)
	}

	reader, writer, _ := os.Pipe()
	go func() {
		defer writer.Close()
		writer.Write(buff.Bytes())
	}()
	content := directory.NewFileContent(file, directory.FromFileObj(reader))

	return content, nil
}

func (r *S3DirectoryRepository) listen(events <-chan directory.Event, errors chan<- error) {
	for {
		select {
		case evt, ok := <-events:
			if !ok {
				return
			}
			ctx := evt.Context()

			s, err := r.getSession(ctx, evt.ConnectionID())
			if err != nil {
				errors <- err
				continue
			}

			switch evt.Name() {
			case directory.CreatedEventName:
				if err := r.handleDirectoryCreation(ctx, s, evt.(directory.DirectoryEvent)); err != nil {
					evt.CallErrorCallbacks(err)
					errors <- fmt.Errorf("failed creating directory: %w", err)
				}
				evt.CallSuccessCallbacks()

			case directory.DeletedEventName:
				errors <- fmt.Errorf("deleting directories is not yet implemented")

			case directory.FileCreatedEventName:
				errors <- fmt.Errorf("file creation is not yet implemented")

			case directory.FileDeletedEventName:
				if err := r.handleFileDeletion(ctx, s, evt.(directory.FileEvent)); err != nil {
					evt.CallErrorCallbacks(err)
					errors <- fmt.Errorf("failed deleting file: %w", err)
				}
				evt.CallSuccessCallbacks()

			case directory.ContentUploadedEventName:
				if err := r.handleUpload(ctx, s, evt.(directory.ContentEvent)); err != nil {
					evt.CallErrorCallbacks(err)
					errors <- fmt.Errorf("failed uploading file: %w", err)
				}
				evt.CallSuccessCallbacks()

			case directory.ContentDownloadEventName:
				if err := r.handleDownload(ctx, s, evt.(directory.ContentEvent)); err != nil {
					evt.CallErrorCallbacks(err)
					errors <- fmt.Errorf("failed downloading file: %w", err)
				}
				evt.CallSuccessCallbacks()

			default:
				errors <- fmt.Errorf("unknown event: %s", evt.Name())
			}
		}
	}
}
