package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

var (
	ErrObjectNotExists = errors.New("object does not exist")
)

type s3Client struct {
	log     *zap.SugaredLogger
	session *session.Session
	s3      *s3.S3

	Bucket string
}

func newS3Client(log *zap.SugaredLogger, accessKey, secretKey, server, bucket, region string, useTls bool) (*s3Client, error) {
	if region == "" {
		region = "us-east-1" // for custom endpoints, value is not important but still required
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:    aws.String(server),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			log.Infof(args[0].(string), args[1:]...)
		}),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(!useTls),
	})
	if err != nil {
		log.Errorf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("newS3Client: %w", err)
	}

	return &s3Client{
		Bucket:  bucket,
		session: sess,
		s3:      s3.New(sess),
		log:     log,
	}, nil
}

func (d *s3Client) GetDirectoriesAndFileByPath(ctx context.Context, currDir *explorer.Directory) ([]*explorer.Directory, []*explorer.RemoteFile, error) {
	var queryPath string
	if currDir == explorer.RootDir {
		queryPath = ""
	} else {
		queryPath = strings.TrimPrefix(currDir.Path(), "/") + "/"
	}

	var files = make([]*explorer.RemoteFile, 0)
	var dirs = make([]*explorer.Directory, 0)

	if err := d.s3.ListObjectsPagesWithContext(
		ctx,
		&s3.ListObjectsInput{
			Bucket:    aws.String(d.Bucket),
			Prefix:    aws.String(queryPath),
			Delimiter: aws.String("/"),
			MaxKeys:   aws.Int64(10),
		},
		func(page *s3.ListObjectsOutput, lastPage bool) bool {
			for _, obj := range page.Contents {
				key := *obj.Key
				if key == queryPath {
					continue
				}
				newFile := explorer.NewRemoteFile(key)
				newFile.SetSizeBytes(*obj.Size)
				newFile.SetLastModified(*obj.LastModified)
				files = append(files, newFile)
			}

			for _, obj := range page.CommonPrefixes {
				if *obj.Prefix == queryPath {
					continue
				}
				s3Prefix := *obj.Prefix
				isDir := strings.HasSuffix(s3Prefix, "/")
				if isDir {
					dirPathStriped := strings.TrimSuffix(s3Prefix, "/")
					dirPathSplit := strings.Split(dirPathStriped, "/")
					dirName := dirPathSplit[len(dirPathSplit)-1]
					newDir := explorer.NewDirectory(dirName, currDir)
					dirs = append(dirs, newDir)
				}
			}
			return !lastPage
		},
	); err != nil {
		d.log.Errorf("Error listing objects: %v\n", err)
		return nil, nil, fmt.Errorf("GetDirectoriesAndFileByPath: %w", err)
	}

	return dirs, files, nil
}

func (d *s3Client) GetFileContent(ctx context.Context, file *explorer.RemoteFile) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(d.Bucket),
		Key:    aws.String(file.Path()),
	}

	result, err := d.s3.GetObjectWithContext(ctx, input)
	if err != nil {
		d.log.Errorf("Error getting object: %v\n", err)
		return nil, fmt.Errorf("GetFileContent: %w", err)
	}

	defer result.Body.Close()

	buff := new(bytes.Buffer)
	_, err = buff.ReadFrom(result.Body)
	if err != nil {
		d.log.Errorf("Error reading object: %v\n", err)
		return nil, fmt.Errorf("GetFileContent: %w", err)
	}
	return buff.Bytes(), nil
}

func (d *s3Client) DownloadFile(ctx context.Context, key, dest string) error {
	downloader := s3manager.NewDownloader(d.session)

	file, err := os.Create(dest)
	if err != nil {
		d.log.Errorf("Error creating file: %v\n", err)
		return fmt.Errorf("DownloadFile: %w", err)
	}
	defer file.Close()

	numBytes, err := downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(d.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		d.log.Errorf("Error downloading file: %v\n", err)
		return fmt.Errorf("DownloadFile: %w", err)
	}

	d.log.Infof("Downloaded %s, %d bytes\n", key, numBytes)
	return nil
}

func (d *s3Client) UploadFile(ctx context.Context, local *explorer.LocalFile, remote *explorer.RemoteFile) error {
	file, err := os.Open(local.Path())
	if err != nil {
		d.log.Errorf("Error opening file: %v\n", err)
		return fmt.Errorf("UploadFile: %w", err)
	}
	defer file.Close()

	uploader := s3manager.NewUploader(d.session)
	_, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(d.Bucket),
		Key:    aws.String(remote.Path()),
		Body:   file,
	})
	if err != nil {
		d.log.Errorf("Error uploading file: %v\n", err)
		return fmt.Errorf("UploadFile: %w", err)
	}

	d.log.Infof("Uploaded %s\n", remote.Path())
	return nil
}
