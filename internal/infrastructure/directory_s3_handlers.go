package infrastructure

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func (r *S3DirectoryRepository) handleDirectoryCreation(ctx context.Context, evt directory.CreatedEvent) error {
	sess, err := r.getSession(ctx, evt.Parent().ConnectionID())
	if err != nil {
		return err
	}

	newDir := evt.Directory()
	if newDir == nil {
		return fmt.Errorf("directory path is empty for created event")
	}

	key := mapDirToObjectKey(newDir)
	if _, err := sess.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
		Body:   strings.NewReader(""),
	}); err != nil {
		return r.manageAwsSdkError(err, newDir.Path().String(), sess)
	}

	return nil
}

func (r *S3DirectoryRepository) handleFileDeletion(ctx context.Context, evt directory.FileDeletedEvent) error {
	sess, err := r.getSession(ctx, evt.ConnectionID())
	if err != nil {
		return err
	}

	file := evt.File()
	if file == nil {
		return fmt.Errorf("file is nil for deletion event")
	}

	key := mapFileToKey(file)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
	}

	if _, err := sess.client.DeleteObject(ctx, input); err != nil {
		return r.manageAwsSdkError(err, file.FullPath(), sess)
	}
	return nil
}

func (r *S3DirectoryRepository) handleUpload(ctx context.Context, evt directory.ContentUploadedEvent) error {
	sess, err := r.getSession(ctx, evt.Directory().ConnectionID())
	if err != nil {
		return err
	}

	content := evt.Content()
	if content == nil {
		return fmt.Errorf("content is nil for upload event")
	}

	fileObj, err := content.Open()
	if err != nil {
		return fmt.Errorf("failed opening the file to upload: %w", err)
	}
	defer func(fileObj io.ReadCloser) {
		if err := fileObj.Close(); err != nil {
			logger.Printf("failed closing file: %v", err)
		}
	}(fileObj)

	uploader := s3manager.NewUploader(sess.client)
	if _, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(content.File())),
		Body:   fileObj,
	}); err != nil {
		return r.manageAwsSdkError(err, content.File().FullPath(), sess)
	}

	return nil
}

func (r *S3DirectoryRepository) handleDownload(ctx context.Context, evt directory.ContentDownloadedEvent) error {
	sess, err := r.getSession(ctx, evt.ConnectionID())
	if err != nil {
		return err
	}

	downloader := s3manager.NewDownloader(sess.client)

	file, err := evt.Content().Open()
	if err != nil {
		return fmt.Errorf("failed opening the file to download: %w", err)
	}
	defer file.Close() //nolint:errcheck

	if _, err = downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(evt.Content().File())),
	}); err != nil {
		return r.manageAwsSdkError(err, evt.Content().File().FullPath(), sess)
	}
	return nil
}
