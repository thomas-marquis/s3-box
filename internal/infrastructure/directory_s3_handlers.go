package infrastructure

import (
	"context"
	"fmt"
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

func (r *S3DirectoryRepository) handleUpload(ctx context.Context, evt directory.ContentUploadedEvent) (*directory.File, error) {
	sess, err := r.getSession(ctx, evt.Directory().ConnectionID())
	if err != nil {
		return nil, err
	}

	content := evt.Content()
	if content == nil {
		return nil, fmt.Errorf("content is nil for upload event")
	}

	fileObj, err := content.Open()
	if err != nil {
		return nil, fmt.Errorf("failed opening the file to upload: %w", err)
	}
	defer fileObj.Close() //nolint:errcheck

	info, err := fileObj.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed reading the file info: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("failed opening the file to upload: path is a directory")
	}

	if _, err = sess.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(sess.connection.Bucket()),
		Key:           aws.String(mapFileToKey(content.File())),
		Body:          fileObj,
		ContentLength: aws.Int64(info.Size()),
	}); err != nil {
		return nil, r.manageAwsSdkError(err, content.File().FullPath(), sess)
	}

	uploadedFile, err := directory.NewFile(
		content.File().Name().String(), content.File().DirectoryPath(),
		directory.WithFileSize(int(info.Size())),
		directory.WithFileLastModified(info.ModTime()))
	if err != nil {
		return nil, fmt.Errorf("failed creating uploaded file: %w", err)
	}

	return uploadedFile, nil
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

func (r *S3DirectoryRepository) handleLoading(ctx context.Context, evt directory.LoadEvent) ([]*directory.Directory, []*directory.File, error) {
	dir := evt.Directory()
	searchKey := mapPathToSearchKey(dir.Path())

	s, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		return nil, nil, err
	}

	inputs := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.connection.Bucket()),
		Prefix:    aws.String(searchKey),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1000),
	}

	files := make([]*directory.File, 0)
	subDirectories := make([]*directory.Directory, 0)

	paginator := s3.NewListObjectsV2Paginator(s.client, inputs)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, nil, r.manageAwsSdkError(
				fmt.Errorf("error while fetching next objects page: %w", err),
				searchKey,
				s)
		}

		for _, obj := range page.Contents {
			key := *obj.Key
			if key == searchKey {
				continue
			}
			f, err := directory.NewFile(mapKeyToObjectName(key), dir.Path(),
				directory.WithFileSize(int(*obj.Size)),
				directory.WithFileLastModified(*obj.LastModified))
			if err != nil {
				return nil, nil, fmt.Errorf("error while creating a file: %w", err)
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
				d, err := directory.New(dir.ConnectionID(), directory.NewPath(s3Prefix).DirectoryName(), dir.Path())
				if err != nil {
					return nil, nil, fmt.Errorf("error while loading a directory: %w", err)
				}
				subDirectories = append(subDirectories, d)
			}
		}
	}

	return subDirectories, files, nil
}
