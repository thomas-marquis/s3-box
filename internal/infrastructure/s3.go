package infrastructure

// import (
// 	"bytes"
// 	"context"
// 	"errors"
// 	"fmt"
// 	"github.com/thomas-marquis/s3-box/internal/explorer"
// 	"os"
// 	"strings"

// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/aws/credentials"
// 	"github.com/aws/aws-sdk-go/aws/session"
// 	"github.com/aws/aws-sdk-go/service/s3"
// 	"github.com/aws/aws-sdk-go/service/s3/s3manager"
// 	"go.uber.org/zap"
// )

// var (
// 	ErrObjectNotExists = errors.New("object does not exist")
// )

// type s3Client struct {
// 	log     *zap.SugaredLogger
// }

// func newS3Client(log *zap.SugaredLogger, accessKey, secretKey, server, bucket, region string, useTls bool) (*s3Client, error) {
// 	if region == "" {
// 		region = "us-east-1" // for custom endpoints, value is not important but still required
// 	}
// 	sess, err := session.NewSession(&aws.Config{
// 		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
// 		Endpoint:    aws.String(server),
// 		Logger: aws.LoggerFunc(func(args ...interface{}) {
// 			log.Infof(args[0].(string), args[1:]...)
// 		}),
// 		Region:           aws.String(region),
// 		S3ForcePathStyle: aws.Bool(true),
// 		DisableSSL:       aws.Bool(!useTls),
// 	})
// 	if err != nil {
// 		log.Errorf("Error creating session: %v\n", err)
// 		return nil, fmt.Errorf("newS3Client: %w", err)
// 	}

// 	return &s3Client{
// 		Bucket:  bucket,
// 		session: sess,
// 		s3:      s3.New(sess),
// 		log:     log,
// 	}, nil
// }

// func (d *s3Client) UploadFile(ctx context.Context, local *explorer.LocalFile, remote *explorer.S3File) error {
// 	file, err := os.Open(local.Path())
// 	if err != nil {
// 		d.log.Errorf("Error opening file: %v\n", err)
// 		return fmt.Errorf("UploadFile: %w", err)
// 	}
// 	defer file.Close()

// 	uploader := s3manager.NewUploader(d.session)
// 	_, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
// 		Bucket: aws.String(d.Bucket),
// 		Key:    aws.String(remote.Path()),
// 		Body:   file,
// 	})
// 	if err != nil {
// 		d.log.Errorf("Error uploading file: %v\n", err)
// 		return fmt.Errorf("UploadFile: %w", err)
// 	}

// 	d.log.Infof("Uploaded %s\n", remote.Path())
// 	return nil
// }
