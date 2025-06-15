package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/thomas-marquis/s3-box/internal/connections"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"go.uber.org/zap"
)

type S3FileRepositoryImpl struct {
	log     *zap.SugaredLogger
	session *session.Session
	s3      *s3.S3
	conn    *connections.Connection
}

var _ explorer.S3FileRepository = &S3FileRepositoryImpl{}

func NewS3FileRepository(logger *zap.Logger, conn *connections.Connection) (*S3FileRepositoryImpl, error) {
	log := logger.Sugar()

	region := conn.Region
	if region == "" {
		region = "us-east-1" // for custom endpoints, value is not important but still required
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(conn.AccessKey, conn.SecretKey, ""),
		Endpoint:    aws.String(conn.Server),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			log.Infof(args[0].(string), args[1:]...)
		}),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(!conn.UseTls),
	})
	if err != nil {
		log.Errorf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("NewS3FileRepository(conn=%s): %w", conn.Name, err)
	}

	return &S3FileRepositoryImpl{
		conn:    conn,
		session: sess,
		s3:      s3.New(sess),
		log:     log,
	}, nil
}

func (r *S3FileRepositoryImpl) GetContent(ctx context.Context, id explorer.S3FileID) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.conn.BucketName),
		Key:    aws.String(string(id)),
	}

	result, err := r.s3.GetObjectWithContext(ctx, input)
	if err != nil {
		r.log.Errorf("Error getting object: %v\n", err)
		return nil, fmt.Errorf("GetContent: %w", err)
	}

	defer result.Body.Close()

	buff := new(bytes.Buffer)
	_, err = buff.ReadFrom(result.Body)
	if err != nil {
		r.log.Errorf("Error reading object: %v\n", err)
		return nil, fmt.Errorf("GetContent: %w", err)
	}
	return buff.Bytes(), nil
}

func (r *S3FileRepositoryImpl) DownloadFile(ctx context.Context, key, dest string) error {
	downloader := s3manager.NewDownloader(r.session)

	file, err := os.Create(dest)
	if err != nil {
		r.log.Errorf("Error creating file: %v\n", err)
		return fmt.Errorf("DownloadFile: %w", err)
	}
	defer file.Close()

	numBytes, err := downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(r.conn.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		r.log.Errorf("Error downloading file: %v\n", err)
		return fmt.Errorf("DownloadFile: %w", err)
	}

	r.log.Infof("Downloaded %s, %d bytes\n", key, numBytes)
	return nil
}

func (r *S3FileRepositoryImpl) UploadFile(ctx context.Context, local *explorer.LocalFile, remote *explorer.S3File) error {
	file, err := os.Open(local.Path())
	if err != nil {
		r.log.Errorf("Error opening file: %v\n", err)
		return fmt.Errorf("UploadFile: %w", err)
	}
	defer file.Close()

	uploader := s3manager.NewUploader(r.session)
	_, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(r.conn.BucketName),
		Key:    aws.String(remote.ID.String()),
		Body:   file,
	})
	if err != nil {
		r.log.Errorf("Error uploading file: %v\n", err)
		return fmt.Errorf("UploadFile: %w", err)
	}

	r.log.Infof("Uploaded %s\n", remote.ID.String())
	return nil
}

func (r *S3FileRepositoryImpl) DeleteFile(ctx context.Context, id explorer.S3FileID) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.conn.BucketName),
		Key:    aws.String(string(id)),
	}

	_, err := r.s3.DeleteObjectWithContext(ctx, input)
	if err != nil {
		r.log.Errorf("Error deleting file: %v\n", err)
		return fmt.Errorf("DeleteFile: %w", err)
	}

	r.log.Infof("Deleted %s\n", id)
	return nil
}
