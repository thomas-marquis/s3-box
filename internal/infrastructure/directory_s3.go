package infrastructure

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
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
	connectionRepository FyneConnectionsRepository
	logger               *log.Logger
	cache                map[connection_deck.ConnectionID]*s3Session
}

var _ directory.Repository = &S3DirectoryRepository{}

func NewS3DirectoryRepository(
	connectionsRepository FyneConnectionsRepository,
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
	// Implementation for retrieving a directory by its path
	return nil, nil
}

func (r *S3DirectoryRepository) DownloadFile(ctx context.Context, connID connection_deck.ConnectionID, file *directory.File) (*directory.Content, error) {
	// Implementation for downloading a file from S3
	return nil, nil
}

func (r *S3DirectoryRepository) UploadFile(ctx context.Context, connID connection_deck.ConnectionID, destDir *directory.Directory, content *directory.Content) (*directory.File, error) {
	return nil, nil
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

func (r *S3DirectoryRepository) handleDirectoryCreation(ctx context.Context, sess *s3Session, evt directory.DirectoryEvent) error {
	newDir := evt.Directory()
	if newDir == nil {
		return fmt.Errorf("directory path is empty for created event")
	}
	if _, err := sess.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapDirToKey(newDir)),
		Body:   strings.NewReader(""),
	}); err != nil {
		return fmt.Errorf("failed to save empty directory: %w", err)
	}

	return nil
}

func (r *S3DirectoryRepository) handleFileDeletion(ctx context.Context, sess *s3Session, evt directory.FileEvent) error {
	file := evt.File()
	if file == nil {
		return fmt.Errorf("file is nil for deletion event")
	}
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(file)),
	}

	if _, err := sess.client.DeleteObjectWithContext(ctx, input); err != nil {
		return fmt.Errorf("failed deleting file: %w", err)
	}
	return nil
}

func (r *S3DirectoryRepository) handleUpload(ctx context.Context, sess *s3Session, evt directory.ContentEvent) error {
	content := evt.Content()
	if content == nil {
		return fmt.Errorf("content is nil for upload event")
	}

	fileObj, err := content.Open()
	if err != nil {
		return fmt.Errorf("failed opening the file to upload: %w", err)
	}
	defer func(fileObj *os.File) {
		if err := fileObj.Close(); err != nil {
			logger.Printf("failed closing file: %v", err)
		}
	}(fileObj)

	uploader := s3manager.NewUploader(sess.session)
	if _, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(content.File())),
		Body:   fileObj,
	}); err != nil {
		return fmt.Errorf("failed uploading file: %w", err)
	}

	return nil
}

func (r *S3DirectoryRepository) handleDownload(ctx context.Context, sess *s3Session, evt directory.ContentEvent) error {
	downloader := s3manager.NewDownloader(sess.session)

	file, err := evt.Content().Open()
	defer file.Close()

	_, err = downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(evt.Content().File())),
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *S3DirectoryRepository) getSession(ctx context.Context, id connection_deck.ConnectionID) (*s3Session, error) {
	conn, err := r.connectionRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if found := r.getFromCache(conn); found != nil {
		return found, nil
	}

	region := conn.Region()
	if region == "" {
		region = "us-east-1" // for custom endpoints, value is not important but still required
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(conn.AccessKey(), conn.SecretKey(), ""),
		Endpoint:    aws.String(conn.Server()),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			logger.Printf(args[0].(string), args[1:]...)
		}),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(!conn.IsTLSActivated()),
	})
	if err != nil {
		r.logger.Printf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("NewS3Repository(conn=%s): %w", conn.Name(), err)
	}
	s := &s3Session{
		session:    sess,
		client:     s3.New(sess),
		connection: conn,
	}
	r.Lock()
	defer r.Unlock()
	r.cache[conn.ID()] = s
	return s, nil
}

func (r *S3DirectoryRepository) getFromCache(c *connection_deck.Connection) *s3Session {
	r.Lock()
	defer r.Unlock()

	found, ok := r.cache[c.ID()]
	if ok && found != nil && found.connection.Is(c) {
		return found
	}
	return nil
}

func mapDirToKey(dir *directory.Directory) string {
	if dir.Path().String() == "" {
		return ""
	}
	return strings.TrimPrefix(dir.Path().String(), "/")
}

func mapFileToKey(file *directory.File) string {
	return strings.TrimPrefix(file.FullPath(), "/")
}
