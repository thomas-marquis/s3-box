package s3

import (
	"fmt"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func (h *EventHandler) handleDeleteFile(evt event.Event) {
	ctx := evt.Context()
	pl := evt.Payload().(directory.DeleteFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
		h.bus.Publish(evt.NewFollowup(
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
		evt.NewFollowup(directory.DeleteFileSucceeded{File: pl.File, ParentDirectory: pl.ParentDirectory}))
}

func (h *EventHandler) handleDeleteDirectory(evt event.Event) {
	pl := evt.Payload().(directory.DeleteTriggered)
	ctx := evt.Context()

	parent := pl.Directory
	child, err := parent.GetSubDirectoryByName(pl.DeletedDirPath.DirectoryName())
	if err != nil {
		h.notifier.NotifyError(fmt.Errorf("failed deleting directory: %w", err))
		h.bus.Publish(evt.NewFollowup(
			directory.DeleteFailed{Err: err, Parent: pl.Directory}))
		return
	}

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed deleting directory: %w", err))
		h.bus.Publish(evt.NewFollowup(
			directory.DeleteFailed{Err: err, Parent: pl.Directory, Directory: child}))
	}

	client, err := h.clientFactory.Get(ctx, pl.Directory.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	key := mapPathToObjectKey(pl.DeletedDirPath)
	if err := client.DeleteObject(ctx, key); err != nil {
		handleError(err)
		return
	}

	h.bus.Publish(
		evt.NewFollowup(directory.DeleteSucceeded{Directory: child, Parent: pl.Directory}))
}
