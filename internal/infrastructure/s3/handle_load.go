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

func (r *EventHandler) handleLoadDirectory(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.LoadEvent)
	dir := evt.Directory()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed loading directory: %w", err))
		r.bus.Publish(directory.NewLoadFailureEvent(err, dir))
	}

	client, err := r.clientFactory.Get(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	if err := r.loadDirectory(ctx, client, dir); err != nil {
		handleError(err)
	}
}

func (r *EventHandler) loadDirectory(ctx context.Context, client s3client.Client, dir *directory.Directory) error {
	searchKey := mapPathToSearchKey(dir.Path())

	files := make([]*directory.File, 0)
	subDirectories := make([]*directory.Directory, 0)

	if err := client.ListObjectsWithCallback(ctx, searchKey, false, func(page *s3.ListObjectsV2Output) error {
		for _, obj := range page.Contents {
			key := *obj.Key

			if isRenameMarkerFile(key) {
				return r.getPendingRenameErr(ctx, client, dir, key)
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

	r.bus.Publish(directory.NewLoadSuccessEvent(dir, subDirectories, files))
	return nil
}

func (r *EventHandler) handleLoadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.FileLoadEvent)
	obj, err := r.loadFile(ctx, evt.File(), evt.ConnectionID())
	if err != nil {
		r.notifier.NotifyError(fmt.Errorf("failed loading file: %w", err))
		r.bus.Publish(directory.NewFileLoadFailureEvent(err, evt.File()))
		return
	}
	r.bus.Publish(directory.NewFileLoadSuccessEvent(evt.File(), obj))
}

func (r *EventHandler) loadFile(ctx context.Context, file *directory.File, connID connection_deck.ConnectionID) (directory.FileContent, error) {
	client, err := r.clientFactory.Get(ctx, connID)
	if err != nil {
		return nil, err
	}
	return NewObject(ctx, client, file)
}
