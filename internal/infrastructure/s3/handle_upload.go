package s3

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
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
