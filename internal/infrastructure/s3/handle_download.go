package s3

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
)

func (h *EventHandler) handleDownloadFile(e event.Event) {
	ctx := e.Context()
	pl := e.Payload().(directory.DownloadFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
		h.bus.Publish(e.NewFollowup(directory.DownloadFileFailed{Err: err}))
	}

	client, err := h.clientFactory.Get(ctx, pl.ConnectionID)
	if err != nil {
		handleError(err)
		return
	}

	localFile, err := os.OpenFile(pl.DstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		handleError(fmt.Errorf("failed opening the file to download: %w", err))
		return
	}
	defer u.SkipD(localFile.Close)

	if err := client.Download(ctx, mapFileToKey(pl.File), localFile); err != nil {
		handleError(fmt.Errorf("failed downloading file: %w", err))
		return
	}

	h.bus.Publish(e.NewFollowup(directory.DownloadFileSucceeded{File: pl.File}))
}

func (h *EventHandler) handleDownload(e event.Event) {
	ctx := e.Context()
	pl := e.Payload().(directory.DownloadTriggered)
	dir := pl.Directory
	destParent := pl.DestParentPath

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed downloading directory: %w", err))
		h.bus.Publish(e.NewFollowup(directory.DownloadFailed{Directory: dir, Err: err}))
	}

	fi, err := os.Stat(destParent)
	if err != nil {
		handleError(fmt.Errorf("failed to stat destination directory: %w", err))
		return
	}
	if !fi.IsDir() {
		handleError(errors.New("destination must be a directory"))
		return
	}

	destPath := filepath.Join(pl.DestParentPath, dir.Name())
	fi, err = os.Stat(destPath)
	if err == nil && fi.IsDir() {
		handleError(fmt.Errorf("destination directory already exists"))
		return
	}

	if err := os.MkdirAll(destPath, 0755); err != nil {
		handleError(fmt.Errorf("failed to create destination directory: %w", err))
		return
	}

	client, err := h.clientFactory.Get(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	if err := client.DownloadDirectory(ctx, mapPathToSearchKey(dir.Path()), destPath); err != nil {
		handleError(fmt.Errorf("failed downloading directory: %w", err))
		return
	}

	h.bus.Publish(e.NewFollowup(directory.DownloadSucceeded{Directory: dir}))
}
