package s3

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *RepositoryImpl) handleDownloadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.ContentDownloadedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
		r.bus.Publish(directory.NewContentDownloadedFailureEvent(err))
	}

	client, err := r.clientFactory.Get(ctx, evt.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	file, err := evt.Content().Open()
	if err != nil {
		handleError(fmt.Errorf("failed opening the file to download: %w", err))
		return
	}
	defer file.Close() //nolint:errcheck

	if err := client.Download(ctx, mapFileToKey(evt.Content().File()), file); err != nil {
		handleError(fmt.Errorf("failed downloading file: %w", err))
		return
	}

	r.bus.Publish(directory.NewContentDownloadedSuccessEvent(evt.Content()))
}
