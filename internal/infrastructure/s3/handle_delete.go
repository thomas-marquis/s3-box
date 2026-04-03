package s3

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (h *EventHandler) handleDeleteFile(evt event.Event) {
	ctx := evt.Context()
	e := evt.(directory.FileDeletedEvent)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
		h.bus.Publish(e.NewFailureEvent(err))
	}

	client, err := h.clientFactory.Get(ctx, e.ConnectionID)
	if err != nil {
		handleError(err)
		return
	}

	file := e.File
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

	h.bus.Publish(e.NewSuccessEvent(e.File))
}

func (h *EventHandler) handleDeleteDirectory(evt event.Event) {
	e := evt.(directory.DeletedEvent)
	err := fmt.Errorf("deleting directories is not yet implemented")
	h.notifier.NotifyError(err)
	h.bus.Publish(e.NewFailureEvent(err))
}
