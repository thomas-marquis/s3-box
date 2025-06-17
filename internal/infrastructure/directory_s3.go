package infrastructure

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type S3DirectoryRepository struct {
	session *session.Session
	s3      *s3.S3
	bucket  string
}

var _ directory.Repository = &S3DirectoryRepository{}

func NewS3Repository(conn *connections.Connection) (*S3DirectoryRepository, error) {
	logger = log.New(os.Stdout, "S3Repository: ", log.LstdFlags)

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
		DisableSSL:       aws.Bool(!conn.UseTLS()),
	})
	if err != nil {
		log.Printf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("NewS3Repository(conn=%s): %w", conn.Name(), err)
	}
	return &S3DirectoryRepository{
		session: sess,
		s3:      s3.New(sess),
		bucket:  conn.Bucket(),
	}, nil
}

func (r *S3DirectoryRepository) GetByPath(ctx context.Context, path directory.Path) (*directory.Directory, error) {
	// Implementation for retrieving a directory by its path
	return nil, nil
}

func (r *S3DirectoryRepository) Save(ctx context.Context, dir *directory.Directory) error {
	// Implementation for saving a directory
	return nil
}

func (r *S3DirectoryRepository) Delete(ctx context.Context, Path directory.Path) error {
	// Implementation for deleting a directory by its path
	return nil
}

func (r *S3DirectoryRepository) DownloadFile(ctx context.Context, file *directory.File, destPath string) error {
	// Implementation for downloading a file from S3
	return nil
}

func (r *S3DirectoryRepository) UploadFile(ctx context.Context, srcPath string, destFile *directory.File) error {
	// Implementation for uploading a file to S3
	return nil
}

func (r *S3DirectoryRepository) LoadContent(ctx context.Context, file *directory.File) ([]byte, error) {
	// Implementation for loading content of a file from S3
	return nil, nil
}
