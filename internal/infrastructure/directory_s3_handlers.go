package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *S3DirectoryRepository) handleCreateDirectory(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.CreatedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed creating directory: %w", err))
		r.bus.Publish(directory.NewCreatedFailureEvent(err, evt.Parent()))
	}

	sess, err := r.getSession(ctx, evt.Parent().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	newDir := evt.Directory()
	if newDir == nil {
		handleError(fmt.Errorf("directory path is empty for created event"))
		return
	}

	key := mapDirToObjectKey(newDir)
	if _, err := sess.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
		Body:   strings.NewReader(""),
	}); err != nil {
		handleError(r.manageAwsSdkError(err, newDir.Path().String(), sess))
		return
	}

	r.bus.Publish(
		directory.NewCreatedSuccessEvent(evt.Parent(), evt.Directory()))
}

func (r *S3DirectoryRepository) handleDeleteDirectory(_ event.Event) {
	err := fmt.Errorf("deleting directories is not yet implemented")
	r.notifier.NotifyError(err)
	r.bus.Publish(directory.NewDeletedFailureEvent(err))
}

func (r *S3DirectoryRepository) handleDeleteFile(evt event.Event) {
	ctx := evt.Context()
	e := evt.(directory.FileDeletedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
		r.bus.Publish(directory.NewFileDeletedFailureEvent(err, e.Parent()))
	}

	sess, err := r.getSession(ctx, e.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	file := e.File()
	if file == nil {
		err := fmt.Errorf("file is nil for deletion event")
		handleError(err)
		return
	}

	key := mapFileToKey(file)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
	}

	if _, err := sess.client.DeleteObject(ctx, input); err != nil {
		err := r.manageAwsSdkError(err, file.FullPath(), sess)
		handleError(err)
		return
	}

	r.bus.Publish(
		directory.NewFileDeletedSuccessEvent(e.Parent(), e.File()))
}

func (r *S3DirectoryRepository) handleUploadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.ContentUploadedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed uploading file: %w", err))
		r.bus.Publish(directory.NewContentUploadedFailureEvent(err, evt.Directory()))
	}

	sess, err := r.getSession(ctx, evt.Directory().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	content := evt.Content()
	if content == nil {
		handleError(fmt.Errorf("content is nil for upload event"))
		return
	}

	fileObj, err := content.Open()
	if err != nil {
		handleError(err)
		return
	}
	defer fileObj.Close() //nolint:errcheck

	info, err := fileObj.Stat()
	if err != nil {
		handleError(fmt.Errorf("failed reading the file info: %w", err))
		return
	}
	if info.IsDir() {
		handleError(fmt.Errorf("failed opening the file to upload: path is a directory"))
		return
	}

	if _, err = sess.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(sess.connection.Bucket()),
		Key:           aws.String(mapFileToKey(content.File())),
		Body:          fileObj,
		ContentLength: aws.Int64(info.Size()),
	}); err != nil {
		handleError(r.manageAwsSdkError(err, content.File().FullPath(), sess))
		return
	}

	uploadedFile, err := directory.NewFile(
		content.File().Name().String(), content.File().DirectoryPath(),
		directory.WithFileSize(int(info.Size())),
		directory.WithFileLastModified(info.ModTime()))
	if err != nil {
		handleError(fmt.Errorf("failed creating uploaded file: %w", err))
		return
	}

	r.bus.Publish(directory.NewContentUploadedSuccessEvent(evt.Directory(), uploadedFile))
}

func (r *S3DirectoryRepository) handleDownloadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.ContentDownloadedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
		r.bus.Publish(directory.NewContentDownloadedFailureEvent(err))
	}

	sess, err := r.getSession(ctx, evt.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	downloader := s3manager.NewDownloader(sess.client)

	file, err := evt.Content().Open()
	if err != nil {
		handleError(fmt.Errorf("failed opening the file to download: %w", err))
		return
	}
	defer file.Close() //nolint:errcheck

	if _, err = downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(evt.Content().File())),
	}); err != nil {
		handleError(r.manageAwsSdkError(err, evt.Content().File().FullPath(), sess))
		return
	}

	r.bus.Publish(directory.NewContentDownloadedSuccessEvent(evt.Content()))
}

func (r *S3DirectoryRepository) handleLoading(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.LoadEvent)
	dir := evt.Directory()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed loading directory: %w", err))
		r.bus.Publish(directory.NewLoadFailureEvent(err, dir))
	}

	searchKey := mapPathToSearchKey(dir.Path())

	s, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
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
			handleError(r.manageAwsSdkError(
				fmt.Errorf("error while fetching next objects page: %w", err),
				searchKey,
				s))
			return
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
				handleError(fmt.Errorf("error while creating a file: %w", err))
				return
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
					handleError(fmt.Errorf("error while loading a directory: %w", err))
					return
				}
				subDirectories = append(subDirectories, d)
			}
		}
	}
	r.bus.Publish(directory.NewLoadSuccessEvent(dir, subDirectories, files))
}

func (r *S3DirectoryRepository) handleRenameDirectory(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.RenamedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed renaming directory: %w", err))
		r.bus.Publish(directory.NewRenamedFailureEvent(err, evt.Directory(), evt.OldPath()))
	}

	sess, err := r.getSession(ctx, evt.Directory().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	dir := evt.Directory()
	if dir == nil {
		handleError(fmt.Errorf("directory is nil for rename event"))
		return
	}

	// For rename, we need to copy all objects from old directory to new directory
	// and then delete the old directory
	newDirPath := dir.ParentPath().NewSubPath(evt.NewName())
	newKeyPrefix := mapPathToSearchKey(newDirPath)

	// First, check if target directory already exists
	listInput := &s3.ListObjectsV2Input{
		Bucket:    aws.String(sess.connection.Bucket()),
		Prefix:    aws.String(newKeyPrefix),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1),
	}

	if _, err := sess.client.ListObjectsV2(ctx, listInput); err == nil {
		// Target directory exists
		handleError(fmt.Errorf("target directory %s already exists", newKeyPrefix))
		return
	}

	// Copy all objects from old directory to new directory
	oldKeyPrefix := mapPathToSearchKey(dir.Path())
	copyInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(sess.connection.Bucket()),
		Prefix: aws.String(oldKeyPrefix),
	}

	copyPaginator := s3.NewListObjectsV2Paginator(sess.client, copyInput)
	for copyPaginator.HasMorePages() {
		page, err := copyPaginator.NextPage(ctx)
		if err != nil {
			handleError(r.manageAwsSdkError(err, oldKeyPrefix, sess))
			return
		}

		for _, obj := range page.Contents {
			oldObjKey := *obj.Key
			// Construct new key by replacing old directory prefix with new directory prefix
			newObjKey := strings.Replace(oldObjKey, oldKeyPrefix, newKeyPrefix, 1)

			copySource := sess.connection.Bucket() + "/" + oldObjKey
			_, err := sess.client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     aws.String(sess.connection.Bucket()),
				Key:        aws.String(newObjKey),
				CopySource: aws.String(copySource),
			})
			if err != nil {
				handleError(r.manageAwsSdkError(err, oldObjKey, sess))
				return
			}
		}
	}

	// Delete old directory objects
	delPaginator := s3.NewListObjectsV2Paginator(sess.client, copyInput)
	for delPaginator.HasMorePages() {
		page, err := delPaginator.NextPage(ctx)
		if err != nil {
			handleError(r.manageAwsSdkError(err, oldKeyPrefix, sess))
			return
		}

		// Collect objects to delete
		deleteObjects := make([]s3types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			deleteObjects = append(deleteObjects, s3types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		if len(deleteObjects) > 0 {
			_, err := sess.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(sess.connection.Bucket()),
				Delete: &s3types.Delete{Objects: deleteObjects},
			})
			if err != nil {
				handleError(r.manageAwsSdkError(err, oldKeyPrefix, sess))
				return
			}
		}
	}

	r.bus.Publish(directory.NewRenamedSuccessEvent(dir, evt.OldPath(), evt.NewName()))
}

func (r *S3DirectoryRepository) handleRenameFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.FileRenamedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed renaming file: %w", err))
		r.bus.Publish(directory.NewFileRenamedFailureEvent(err, evt.Parent(), evt.File(), evt.OldName()))
	}

	sess, err := r.getSession(ctx, evt.Parent().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	file := evt.File()
	if file == nil {
		handleError(fmt.Errorf("file is nil for rename event"))
		return
	}

	// Construct old key using the old file name
	oldFile, err := directory.NewFile(string(evt.OldName()), evt.Parent().Path())
	if err != nil {
		handleError(err)
		return
	}
	oldKey := mapFileToKey(oldFile)

	// Construct new key with new filename
	newKey := mapFileToKey(evt.File())

	// Copy the file to new location
	copySource := sess.connection.Bucket() + "/" + oldKey
	_, err = sess.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(sess.connection.Bucket()),
		Key:        aws.String(newKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		handleError(r.manageAwsSdkError(err, oldKey, sess))
		return
	}

	// Delete the old file
	_, err = sess.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		handleError(r.manageAwsSdkError(err, oldKey, sess))
		return
	}

	r.bus.Publish(directory.NewFileRenamedSuccessEvent(evt.Parent(), evt.File(), evt.OldName()))
}

func (r *S3DirectoryRepository) handleCreateFile(evt event.Event) {
	ctx := evt.Context()

	e := evt.(directory.FileCreatedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed creating file: %w", err))
		r.bus.Publish(directory.NewFileCreatedFailureEvent(err, e.Directory()))
	}

	obj, err := r.loadFile(ctx, e.File(), e.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}
	if _, err := obj.Write([]byte{}); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewFileCreatedSuccessEvent(e.Directory(), e.File()))
}

func (r *S3DirectoryRepository) handleLoadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.FileLoadEvent)
	obj, err := r.loadFile(ctx, evt.File(), evt.ConnectionID())
	if err != nil {
		r.notifier.NotifyError(fmt.Errorf("failed loading file: %w", err))
		r.bus.Publish(directory.NewFileLoadFailureEvent(err, evt.File()))
		return
	}
	r.bus.Publish(directory.NewFileLoadSuccessEvent(evt.File(), obj))
}

func (r *S3DirectoryRepository) loadFile(ctx context.Context, file *directory.File, connID connection_deck.ConnectionID) (directory.FileObject, error) {
	sess, err := r.getSession(ctx, connID)
	if err != nil {
		return nil, err
	}
	return NewS3Object(ctx, sess.downloader, sess.uploader, sess.connection, file)
}
