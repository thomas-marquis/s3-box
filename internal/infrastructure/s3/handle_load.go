package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *RepositoryImpl) handleLoadDirectory(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.LoadEvent)
	dir := evt.Directory()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed loading directory: %w", err))
		r.bus.Publish(directory.NewLoadFailureEvent(err, dir))
	}

	searchKey := mapPathToSearchKey(dir.Path())

	s, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	inputs := &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.connection.Bucket()),
		Prefix:    aws.String(searchKey),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1000),
	}

	files := make([]*directory.File, 0)
	subDirectories := make([]*directory.Directory, 0)

	paginator := s3.NewListObjectsV2Paginator(s.client, inputs)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			handleError(r.manageAwsSdkError(
				fmt.Errorf("error while fetching next objects page: %w", err),
				searchKey,
				s))
			return
		}

		for _, obj := range page.Contents {
			key := *obj.Key

			if isRenameMarkerFile(key) {
				handleError(r.getPendingRenameErr(ctx, s, dir, key))
				return
			}

			if key == searchKey {
				continue
			}
			f, err := directory.NewFile(mapKeyToObjectName(key), dir.Path(),
				directory.WithFileSize(int(*obj.Size)),
				directory.WithFileLastModified(*obj.LastModified))
			if err != nil {
				handleError(fmt.Errorf("error while creating a file: %w", err))
				return
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
					handleError(fmt.Errorf("error while loading a directory: %w", err))
					return
				}
				subDirectories = append(subDirectories, d)
			}
		}
	}
	r.bus.Publish(directory.NewLoadSuccessEvent(dir, subDirectories, files))
}

func (r *RepositoryImpl) handleLoadFile(e event.Event) {
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

func (r *RepositoryImpl) loadFile(ctx context.Context, file *directory.File, connID connection_deck.ConnectionID) (directory.FileObject, error) {
	sess, err := r.getSession(ctx, connID)
	if err != nil {
		return nil, err
	}
	return NewObject(ctx, sess.downloader, sess.uploader, sess.connection, file)
}
