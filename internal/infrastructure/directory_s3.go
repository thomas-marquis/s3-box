package infrastructure

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type s3Session struct {
	connection *connections.Connection
	client     *s3.S3
	session    *session.Session
}

type S3DirectoryRepository struct {
	sync.Mutex
	connectionRepository FyneConnectionsRepository
	logger               *log.Logger
	cache                map[connections.ConnectionID]*s3Session
	updates              chan directory.Update
	errors               chan error
}

var _ directory.Repository = &S3DirectoryRepository{}

func NewS3DirectoryRepository(
	connectionsRepository FyneConnectionsRepository,
	updatesStream chan directory.Update,
	errorStream chan error,
) (*S3DirectoryRepository, error) {

	r := &S3DirectoryRepository{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		updates:              updatesStream,
		errors:               errorStream,
	}

	r.listenUpdates()

	return r, nil
}

func (r *S3DirectoryRepository) GetByPath(ctx context.Context, connID connections.ConnectionID, path directory.Path) (*directory.Directory, error) {
	// Implementation for retrieving a directory by its path
	return nil, nil
}

func (r *S3DirectoryRepository) DownloadFile(ctx context.Context, connID connections.ConnectionID, file *directory.File, destPath string) error {
	// Implementation for downloading a file from S3
	return nil
}

func (r *S3DirectoryRepository) LoadContent(ctx context.Context, connID connections.ConnectionID, file *directory.File) ([]byte, error) {
	// Implementation for loading content of a file from S3
	return nil, nil
}

func (r *S3DirectoryRepository) listenUpdates() {
	go func() {
		for {
			select {
			case evt := <-r.updates:
				ctx := context.Background() // TODO: handle timeout here
				if evt.Type() == directory.UpdateTypeNone {
					continue
				}

				dir := evt.Object()
				s, err := r.getSession(ctx, dir.ConnectionID())
				if err != nil {
					r.errors <- err
				}

				switch evt.Type() {
				case directory.UpdateTypeCreated:
					createdDirPath := evt.AttachedDirPath()
					if createdDirPath == "" {
						r.errors <- fmt.Errorf("directory path is empty for created event")
					}
					_, err := s.client.PutObject(&s3.PutObjectInput{
						Bucket: aws.String(s.connection.Bucket()),
						Key:    aws.String(createdDirPath.String()),
						Body:   strings.NewReader(""),
					})
					if err != nil {
						r.errors <- fmt.Errorf("failed to save empty directory: %w", err)
					}
				case directory.UpdateTypeDeleted:
					r.errors <- fmt.Errorf("deleting directories is not yet implemented")
				case directory.UpdateTypeFileCreated:
					r.errors <- fmt.Errorf("file creation is not yest implemented")
				case directory.UpdateTypeFileDeleted:
					file := evt.AttachedFile()
					if file == nil {
						r.errors <- fmt.Errorf("file is nil for deletion event")
					}
					input := &s3.DeleteObjectInput{
						Bucket: aws.String(s.connection.Bucket()),
						Key:    aws.String(file.FullPath()),
					}

					_, err := s.client.DeleteObjectWithContext(ctx, input)
					if err != nil {
						r.errors <- fmt.Errorf("failed deleting file: %w", err)
					}
				default:
					r.errors <- fmt.Errorf("unknown event type: %s", evt.Type())
				}
			}
		}
	}()
}

func (r *S3DirectoryRepository) getSession(ctx context.Context, id connections.ConnectionID) (*s3Session, error) {
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

func (r *S3DirectoryRepository) getFromCache(c *connections.Connection) *s3Session {
	r.Lock()
	defer r.Unlock()
	found, ok := r.cache[c.ID()]
	if ok && found != nil && found.connection.Is(c) {
		return found
	}
	return nil
}

func mapPathToKey(path directory.Path) string {
	if path.String() == "" {
		return ""
	}
	return strings.TrimPrefix(path.String(), "/")
}
