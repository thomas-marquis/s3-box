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

	oldKeyPrefix := mapPathToSearchKey(dir.Path())
	newDirPath := dir.ParentPath().NewSubPath(evt.NewName())
	newKeyPrefix := mapPathToSearchKey(newDirPath)

	// Step 1: Check if target directory already exists and create marker
	markerKey := newKeyPrefix + ".renaming-marker"
	listInput := &s3.ListObjectsV2Input{
		Bucket:    aws.String(sess.connection.Bucket()),
		Prefix:    aws.String(newKeyPrefix),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1),
	}

	result, err := sess.client.ListObjectsV2(ctx, listInput)
	if err != nil {
		handleError(r.manageAwsSdkError(err, newKeyPrefix, sess))
		return
	}

	// Check if target directory actually exists (has objects or common prefixes)
	if len(result.Contents) > 0 || len(result.CommonPrefixes) > 0 {
		// Target directory exists
		handleError(fmt.Errorf("target directory %s already exists", newKeyPrefix))
		return
	}

	// Create marker file to reserve the directory name
	// This prevents other operations from using this directory name during the rename
	_, err = sess.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(markerKey),
		Body:   strings.NewReader("renaming in progress"),
	})
	if err != nil {
		handleError(r.manageAwsSdkError(err, markerKey, sess))
		return
	}

	// Ensure marker is cleaned up on failure
	cleanupMarker := true
	defer func() {
		if cleanupMarker {
			_, _ = sess.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(sess.connection.Bucket()),
				Key:    aws.String(markerKey),
			})
		}
	}()

	// Step 2: Copy objects in batches with progress tracking and error recovery
	copyInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(sess.connection.Bucket()),
		Prefix: aws.String(oldKeyPrefix),
	}

	var copiedObjects []s3types.ObjectIdentifier
	var copyErrors []error
	var totalObjectsCopied int

	copyPaginator := s3.NewListObjectsV2Paginator(sess.client, copyInput)
	for copyPaginator.HasMorePages() {
		page, err := copyPaginator.NextPage(ctx)
		if err != nil {
			copyErrors = append(copyErrors, r.manageAwsSdkError(err, oldKeyPrefix, sess))
			break
		}

		// Process objects in this page
		for _, obj := range page.Contents {
			oldObjKey := *obj.Key
			// Skip the directory marker itself if it exists
			if strings.HasSuffix(oldObjKey, "/") {
				continue
			}

			// Construct new key by replacing old directory prefix with new directory prefix
			newObjKey := strings.Replace(oldObjKey, oldKeyPrefix, newKeyPrefix, 1)

			copySource := sess.connection.Bucket() + "/" + oldObjKey
			_, err := sess.client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     aws.String(sess.connection.Bucket()),
				Key:        aws.String(newObjKey),
				CopySource: aws.String(copySource),
			})
			if err != nil {
				copyErrors = append(copyErrors, r.manageAwsSdkError(err, oldObjKey, sess))
				break
			}

			totalObjectsCopied++
			// Track copied objects for potential cleanup
			copiedObjects = append(copiedObjects, s3types.ObjectIdentifier{
				Key: aws.String(newObjKey),
			})

			// Log progress for large directories
			if totalObjectsCopied%100 == 0 {
				r.logger.Printf("Renamed %d objects from %s to %s", totalObjectsCopied, oldKeyPrefix, newKeyPrefix)
			}
		}

		if len(copyErrors) > 0 {
			break
		}
	}

	// If there were copy errors, clean up what we copied and fail
	if len(copyErrors) > 0 {
		if len(copiedObjects) > 0 {
			r.logger.Printf("Cleaning up %d copied objects due to copy errors", len(copiedObjects))
			_, _ = sess.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(sess.connection.Bucket()),
				Delete: &s3types.Delete{Objects: copiedObjects},
			})
		}
		handleError(fmt.Errorf("failed to copy all objects after %d successful copies: %v", totalObjectsCopied, copyErrors))
		return
	}

	// Step 3: Delete old objects in batches with error recovery
	var deleteErrors []error
	var totalObjectsDeleted int
	delPaginator := s3.NewListObjectsV2Paginator(sess.client, copyInput)
	for delPaginator.HasMorePages() {
		page, err := delPaginator.NextPage(ctx)
		if err != nil {
			deleteErrors = append(deleteErrors, r.manageAwsSdkError(err, oldKeyPrefix, sess))
			break
		}

		// Collect objects to delete (including directory marker objects)
		var deleteObjects []s3types.ObjectIdentifier
		for _, obj := range page.Contents {
			// Include all objects, including directory markers
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
				deleteErrors = append(deleteErrors, r.manageAwsSdkError(err, oldKeyPrefix, sess))
				break
			}
			totalObjectsDeleted += len(deleteObjects)

			// Log progress for large directories
			if totalObjectsDeleted%100 == 0 {
				r.logger.Printf("Deleted %d objects from %s", totalObjectsDeleted, oldKeyPrefix)
			}
		}
	}

	// If there were delete errors, we have an inconsistent state
	// The marker file will remain to indicate the operation failed
	// This allows for manual recovery or retry
	if len(deleteErrors) > 0 {
		cleanupMarker = false // Leave marker to indicate failure
		handleError(fmt.Errorf("failed to delete all old objects after %d successful deletions: %v", totalObjectsDeleted, deleteErrors))
		return
	}

	// Step 4: Clean up marker file (success case)
	cleanupMarker = false // Don't cleanup in defer
	_, err = sess.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(markerKey),
	})
	if err != nil {
		// Log warning but don't fail the operation
		r.logger.Printf("Warning: failed to clean up marker file %s: %v", markerKey, err)
	}

	r.logger.Printf("Successfully renamed directory: %d objects copied, %d objects deleted", totalObjectsCopied, totalObjectsDeleted)
	r.bus.Publish(directory.NewRenamedSuccessEvent(dir, evt.OldPath(), evt.NewName()))

	// Trigger reload of the renamed directory to repopulate its contents
	loadEvt, err := dir.Load()
	if err != nil {
		r.logger.Printf("Warning: failed to trigger reload of renamed directory %s: %v", dir.Path(), err)
	} else {
		r.bus.Publish(loadEvt)
	}
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
