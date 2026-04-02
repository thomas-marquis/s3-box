package s3

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *EventHandler) handleDeleteFile(evt event.Event) {
	ctx := evt.Context()
	e := evt.(directory.FileDeletedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
		r.bus.Publish(directory.NewFileDeletedFailureEvent(err, e.Parent()))
	}

	client, err := r.clientFactory.Get(ctx, e.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	file := e.File()
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

	r.bus.Publish(directory.NewFileDeletedSuccessEvent(e.Parent(), e.File()))
}

func (r *EventHandler) handleDeleteDirectory(_ event.Event) {
	err := fmt.Errorf("deleting directories is not yet implemented")
	r.notifier.NotifyError(err)
	r.bus.Publish(directory.NewDeletedFailureEvent(err))
}
