package s3

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
)

func (r *RepositoryImpl) handleUploadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.ContentUploadedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed uploading file: %w", err))
		r.bus.Publish(directory.NewContentUploadedFailureEvent(err, evt.Directory()))
	}

	client, err := r.clientFactory.Get(ctx, evt.Directory().ConnectionID())
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

	if err := client.PutObject(ctx, mapFileToKey(content.File()), fileObj, s3client.WithContentLength(info.Size())); err != nil {
		handleError(err)
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
