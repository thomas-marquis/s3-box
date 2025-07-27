package infrastructure

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"os"
	"strings"
)

func (r *S3DirectoryRepository) handleDirectoryCreation(ctx context.Context, sess *s3Session, evt directory.DirectoryEvent) error {
	newDir := evt.Directory()
	if newDir == nil {
		return fmt.Errorf("directory path is empty for created event")
	}

	key := mapDirToObjectKey(newDir)
	if _, err := sess.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
		Body:   strings.NewReader(""),
	}); err != nil {
		return r.manageAwsSdkError(err, newDir.Path().String(), sess)
	}

	return nil
}

func (r *S3DirectoryRepository) handleFileDeletion(ctx context.Context, sess *s3Session, evt directory.FileEvent) error {
	file := evt.File()
	if file == nil {
		return fmt.Errorf("file is nil for deletion event")
	}

	key := mapFileToKey(file)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
	}

	if _, err := sess.client.DeleteObjectWithContext(ctx, input); err != nil {
		return r.manageAwsSdkError(err, file.FullPath(), sess)
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
		return r.manageAwsSdkError(err, content.File().FullPath(), sess)
	}

	return nil
}

func (r *S3DirectoryRepository) handleDownload(ctx context.Context, sess *s3Session, evt directory.ContentEvent) error {
	downloader := s3manager.NewDownloader(sess.session)

	file, err := evt.Content().Open()
	defer file.Close()

	if _, err = downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(evt.Content().File())),
	}); err != nil {
		return r.manageAwsSdkError(err, evt.Content().File().FullPath(), sess)
	}
	return nil
}
