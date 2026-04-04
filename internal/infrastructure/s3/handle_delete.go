package s3

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (h *EventHandler) handleDeleteFile(evt event.Event) {
	ctx := evt.Context
	pl := evt.Payload.(directory.DeleteFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
		h.bus.Publish(event.NewFollowup(evt,
			directory.DeleteFileFailed{Err: err, ParentDirectory: pl.ParentDirectory}))
	}

	client, err := h.clientFactory.Get(ctx, pl.ConnectionID)
	if err != nil {
		handleError(err)
		return
	}

	file := pl.File
	if file == nil {
		err := fmt.Errorf("file is nil for deletion event")
		handleError(err)
		return
	}

	key := mapFileToKey(file)
	if err := client.DeleteObject(ctx, key); err != nil {
		handleError(err)
		return
	}

	h.bus.Publish(
		event.NewFollowup(evt, directory.DeleteFileSucceeded{File: pl.File, ParentDirectory: pl.ParentDirectory}))
}

func (h *EventHandler) handleDeleteDirectory(evt event.Event) {
	err := fmt.Errorf("deleting directories is not yet implemented")
	h.notifier.NotifyError(err)
	h.bus.Publish(event.NewFollowup(evt, directory.DeleteFailed{
		Err: err,
	}))
}
