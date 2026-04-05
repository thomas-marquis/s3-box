package s3

import (
	"fmt"
	"strings"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (h *EventHandler) handleCreateFile(evt event.Event) {
	ctx := evt.Context

	e := evt.Payload.(directory.CreateFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed creating file: %w", err))
		h.bus.Publish(
			event.NewFollowup(evt, directory.CreateFileFailed{Err: err, Directory: e.Directory}))
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

	h.bus.Publish(
		event.NewFollowup(evt, directory.CreateFileSucceeded{File: e.File, Directory: e.Directory}))
}

func (h *EventHandler) handleCreateDirectory(e event.Event) {
	ctx := e.Context
	pl := e.Payload.(directory.CreateTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed creating directory: %w", err))
		h.bus.Publish(
			event.NewFollowup(e, directory.CreateFailed{Err: err, ParentDirectory: pl.ParentDirectory}))
	}

	client, err := h.clientFactory.Get(ctx, pl.ParentDirectory.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	newDir := pl.Directory
	if newDir == nil {
		handleError(fmt.Errorf("directory path is empty for created event"))
		return
	}

	key := mapDirToObjectKey(newDir)
	if err := client.PutObject(ctx, key, strings.NewReader("")); err != nil {
		handleError(err)
		return
	}

	h.bus.Publish(
		event.NewFollowup(e, directory.CreateSucceeded{ParentDirectory: pl.ParentDirectory, Directory: newDir}))
}
