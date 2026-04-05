package s3

import (
	"fmt"
	"os"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (h *EventHandler) handleDownloadFile(e event.Event) {
	ctx := e.Context
	pl := e.Payload.(directory.DownloadFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
		h.bus.Publish(event.NewFollowup(e, directory.DownloadFileFailed{Err: err}))
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
	defer localFile.Close() //nolint:errcheck

	if err := client.Download(ctx, mapFileToKey(pl.File), localFile); err != nil {
		handleError(fmt.Errorf("failed downloading file: %w", err))
		return
	}

	h.bus.Publish(event.NewFollowup(e, directory.DownloadFileSucceeded{File: pl.File}))
}
