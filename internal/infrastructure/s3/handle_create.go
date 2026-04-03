package s3

import (
	"fmt"
	"strings"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (h *EventHandler) handleCreateFile(evt event.Event) {
	ctx := evt.Context()

	e := evt.(directory.FileCreatedEvent)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed creating file: %w", err))
		h.bus.Publish(directory.NewFileCreatedFailureEvent(err, e.Directory))
	}

	obj, err := h.loadFile(ctx, e.File, e.ConnectionID)
	if err != nil {
		handleError(err)
		return
	}
	if _, err := obj.Write([]byte{}); err != nil {
		handleError(err)
		return
	}

	h.bus.Publish(directory.NewFileCreatedSuccessEvent(e.Directory, e.File))
}

func (h *EventHandler) handleCreateDirectory(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.CreatedEvent)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed creating directory: %w", err))
		h.bus.Publish(evt.NewFailureEvent(err))
	}

	client, err := h.clientFactory.Get(ctx, evt.ParentDirectory.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	newDir := evt.Directory
	if newDir == nil {
		handleError(fmt.Errorf("directory path is empty for created event"))
		return
	}

	key := mapDirToObjectKey(newDir)
	if err := client.PutObject(ctx, key, strings.NewReader("")); err != nil {
		handleError(err)
		return
	}

	h.bus.Publish(evt.NewSuccessEvent())
}
