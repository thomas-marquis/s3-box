package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
)

func (h *EventHandler) handleUploadFile(e event.Event) {
	ctx := e.Context
	pl := e.Payload.(directory.UploadFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed uploading file: %w", err))
		h.bus.Publish(event.NewFollowup(e, directory.UploadFileFailed{Err: err, Directory: pl.Directory}))
	}

	client, err := h.clientFactory.Get(ctx, pl.Directory.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	localFile, err := os.Open(pl.SrcPath)
	if err != nil {
		handleError(err)
		return
	}
	defer localFile.Close() //nolint:errcheck

	info, err := localFile.Stat()
	if err != nil {
		handleError(fmt.Errorf("failed reading the file info: %w", err))
		return
	}
	if info.IsDir() {
		handleError(fmt.Errorf("failed opening the file to upload: path is a directory"))
		return
	}

	fileName := filepath.Base(pl.SrcPath)
	newFile, err := directory.NewFile(fileName, pl.Directory,
		directory.WithFileSize(int(info.Size())),
		directory.WithFileLastModified(info.ModTime()))
	if err != nil {
		handleError(err)
		return
	}

	if err := client.Upload(ctx, mapFileToKey(newFile), localFile); err != nil {
		handleError(fmt.Errorf("failed uploading file: %w", err))
		return
	}

	h.bus.Publish(event.NewFollowup(e, directory.UploadFileSucceeded{File: newFile, Directory: pl.Directory}))
}

func (h *EventHandler) handleUploadTriggered(e event.Event) {
	pl := e.Payload.(directory.UploadTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("upload failed: %w", err))
		h.bus.Publish(event.NewFollowup(e, directory.UploadFailed{Directory: pl.Directory, Err: err}))
	}

	if len(pl.Items) == 1 && !pl.Items[0].IsDir {
		if err := h.doUpload(e.Context, pl.Directory, pl.Items); err != nil {
			handleError(err)
		}
		return
	}
}

func (h *EventHandler) handleUpload(e event.Event) {
	pl := e.Payload.(directory.UploadConfirmed)
}

func (h *EventHandler) doUpload(ctx context.Context, dir *directory.Directory, items []directory.FsItem) error {
	client, err := h.clientFactory.Get(ctx, dir.ConnectionID())
	if err != nil {
		return err
	}

	return nil
}

func (h *EventHandler) uploadItem(ctx context.Context, client s3client.Client, dir *directory.Directory, item directory.FsItem) error {
	if item.IsDir {
		// upload event:
		if err := h.createEmptyDirectory(ctx, client, "TODO"); err != nil {
			return err
		}
		// Then, upload children items recursively
	}

	return nil
}

// Items:
// f1
// f2
// d1/
// d1/f3
// d1/f4
// d2/
// d2/f5

// processing order:
// [
// upload f1
// upload f2
// create empty dir d1
// create empty dir d2
// ]
// upload f3 into d1
// upload f4 into d1
// upload f5 into d2
// upload f6 into d2
