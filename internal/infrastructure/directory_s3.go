package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
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
	bus event.Bus,
	notifier notification.Repository,
) (*S3DirectoryRepository, error) {
	r := &S3DirectoryRepository{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		cache:                make(map[connection_deck.ConnectionID]*s3Session),
	}

	events := bus.Subscribe()
	for i := 0; i < nbWorkers; i++ {
		go r.listen(events, bus.Publish, notifier)
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

	var dir directory.Directory

	files := make([]*directory.File, 0)
	subDirectoriesPaths := make([]directory.Path, 0)

	pageHandler := func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			key := *obj.Key
			if key == searchKey {
				continue
			}
			f, err := directory.NewFile(mapKeyToObjectName(key), &dir,
				directory.WithFileSize(int(*obj.Size)),
				directory.WithFileLastModified(*obj.LastModified),
			)
			if err != nil {
				return false
			}
			files = append(files, f)
		}

		for _, obj := range page.CommonPrefixes {
			if *obj.Prefix == searchKey {
				continue
			}
			s3Prefix := *obj.Prefix
			isDir := strings.HasSuffix(s3Prefix, "/")
			if isDir {
				subPath := directory.NewPath(mapKeyToObjectName(s3Prefix))
				subDirectoriesPaths = append(subDirectoriesPaths, subPath)
			}
		}
		return !lastPage
	}

	if err := s.client.ListObjectsPagesWithContext(ctx, inputs, pageHandler); err != nil {
		return nil, r.manageAwsSdkError(err, searchKey, s)
	}

	temp, err := directory.New(connID, path.DirectoryName(), path.ParentPath(),
		directory.WithFiles(files),
		directory.WithSubDirectories(subDirectoriesPaths))
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	dir = *temp

	return &dir, nil
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

	rd, w, _ := os.Pipe()
	if _, err := w.Write(buff.Bytes()); err != nil {
		return nil, fmt.Errorf("fail writing the body content: %w", err)
	}
	defer w.Close()

	content := directory.NewFileContent(file, directory.ContentFromFile(rd))

	return content, nil
}

func (r *S3DirectoryRepository) listen(events <-chan event.Event, publisher func(event.Event), notifier notification.Repository) {
	for evt := range events {
		ctx := evt.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		switch evt.Type() {
		case directory.CreatedEventType:
			e := evt.(directory.CreatedEvent)
			if err := r.handleDirectoryCreation(ctx, e); err != nil {
				notifier.NotifyError(fmt.Errorf("failed creating directory: %w", err))
				publisher(directory.NewCreatedFailureEvent(err, e.Parent()))
			}
			publisher(directory.NewCreatedSuccessEvent(e.Parent(), e.Directory()))

		case directory.DeletedEventType:
			err := fmt.Errorf("deleting directories is not yet implemented")
			notifier.NotifyError(err)
			publisher(directory.NewDeletedFailureEvent(err))

		case directory.FileCreatedEventType:
			err := fmt.Errorf("file creation is not yet implemented")
			notifier.NotifyError(err)
			publisher(directory.NewFileCreatedFailureEvent(err))

		case directory.FileDeletedEventType:
			e := evt.(directory.FileDeletedEvent)
			if err := r.handleFileDeletion(ctx, e); err != nil {
				notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
				publisher(directory.NewFileDeletedFailureEvent(err, e.Parent()))
			}
			publisher(directory.NewFileDeletedSuccessEvent(e.Parent(), e.File()))

		case directory.ContentUploadedEventType:
			e := evt.(directory.ContentUploadedEvent)
			if err := r.handleUpload(ctx, e); err != nil {
				notifier.NotifyError(fmt.Errorf("failed uploading file: %w", err))
				publisher(directory.NewContentUploadedFailureEvent(err, e.Directory()))
			}
			publisher(directory.NewContentUploadedSuccessEvent(e.Directory(), e.Content()))

		case directory.ContentDownloadEventType:
			e := evt.(directory.ContentDownloadedEvent)
			if err := r.handleDownload(ctx, e); err != nil {
				notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
				publisher(directory.NewContentDownloadedFailureEvent(err))
			}
			publisher(directory.NewContentDownloadedSuccessEvent(e.Content()))
		}
	}
}
