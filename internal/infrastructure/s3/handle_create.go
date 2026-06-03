package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
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

	if err := h.createEmptyDirectory(ctx, client, newDir.Path()); err != nil {
		handleError(err)
		return
	}

	h.bus.Publish(
		event.NewFollowup(e, directory.CreateSucceeded{ParentDirectory: pl.ParentDirectory, Directory: newDir}))
}

func (h *EventHandler) createEmptyDirectory(ctx context.Context, client s3client.Client, path directory.Path) error {
	key := mapPathToObjectKey(path)
	if err := client.PutObject(ctx, key, strings.NewReader("")); err != nil {
		return err
	}
	return nil
}
