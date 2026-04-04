package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
)

func (h *EventHandler) handleLoadDirectory(e event.Event) {
	ctx := e.Context
	pl := e.Payload.(directory.LoadTriggered)
	dir := pl.Directory

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed loading directory: %w", err))
		h.bus.Publish(event.New(directory.LoadFailed{
			Err:       err,
			Directory: pl.Directory,
		}))
	}

	client, err := h.clientFactory.Get(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	if err := h.loadDirectory(ctx, client, dir, e); err != nil {
		handleError(err)
	}
}

func (h *EventHandler) loadDirectory(ctx context.Context, client s3client.Client, dir *directory.Directory, prevEvent event.Event) error {
	searchKey := mapPathToSearchKey(dir.Path())

	files := make([]*directory.File, 0)
	subDirectories := make([]*directory.Directory, 0)

	if err := client.ListObjectsWithCallback(ctx, searchKey, false, func(page *s3.ListObjectsV2Output) error {
		for _, obj := range page.Contents {
			key := *obj.Key

			if isRenameMarkerFile(key) {
				return h.getPendingRenameErr(ctx, client, dir, key)
			}

			if key == searchKey {
				continue
			}
			f, err := directory.NewFile(mapKeyToObjectName(key), dir,
				directory.WithFileSize(int(*obj.Size)),
				directory.WithFileLastModified(*obj.LastModified))
			if err != nil {
				return fmt.Errorf("error while creating a file: %w", err)
			}
			files = append(files, f)
		}

		for _, obj := range page.CommonPrefixes {
			if *obj.Prefix == searchKey {
				continue
			}
			s3Prefix := *obj.Prefix
			isDir := strings.HasSuffix(s3Prefix, "/")
			if isDir {
				d, err := directory.New(dir.ConnectionID(), directory.NewPath(s3Prefix).DirectoryName(), dir)
				if err != nil {
					return fmt.Errorf("error while loading a directory: %w", err)
				}
				subDirectories = append(subDirectories, d)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	h.bus.Publish(event.NewFollowup(prevEvent, directory.LoadSucceeded{
		Directory:      dir,
		Files:          files,
		SubDirectories: subDirectories,
	}))
	return nil
}

func (h *EventHandler) handleLoadFile(e event.Event) {
	ctx := e.Context
	pl := e.Payload.(directory.LoadFileTriggered)
	obj, err := h.loadFile(ctx, pl.File, pl.ConnectionID)
	if err != nil {
		h.notifier.NotifyError(fmt.Errorf("failed loading file: %w", err))
		h.bus.Publish(event.NewFollowup(e, directory.LoadFileFailed{
			Err:  err,
			File: pl.File,
		}))
		return
	}
	h.bus.Publish(event.NewFollowup(e, directory.LoadFileSucceeded{
		File:    pl.File,
		Content: obj,
	}))
}

func (h *EventHandler) loadFile(ctx context.Context, file *directory.File, connID connection_deck.ConnectionID) (directory.FileContent, error) {
	client, err := h.clientFactory.Get(ctx, connID)
	if err != nil {
		return nil, err
	}
	return NewObject(ctx, client, file)
}
