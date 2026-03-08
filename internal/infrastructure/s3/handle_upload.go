package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *RepositoryImpl) handleUploadFile(e event.Event) {
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
