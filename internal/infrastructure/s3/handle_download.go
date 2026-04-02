package s3

import (
	"fmt"
	"os"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *EventHandler) handleDownloadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.FileDownloadEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
		r.bus.Publish(directory.NewFileDownloadFailureEvent(err))
	}

	client, err := r.clientFactory.Get(ctx, evt.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	localFile, err := os.OpenFile(evt.DstPath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		handleError(fmt.Errorf("failed opening the file to download: %w", err))
		return
	}
	defer localFile.Close() //nolint:errcheck

	if err := client.Download(ctx, mapFileToKey(evt.File()), localFile); err != nil {
		handleError(fmt.Errorf("failed downloading file: %w", err))
		return
	}

	r.bus.Publish(directory.NewFileDownloadSuccessEvent(evt.File()))
}
