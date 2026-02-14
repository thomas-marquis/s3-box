package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

const (
	nbWorkers = 5
)

type s3Session struct {
	connection *connection_deck.Connection
	client     *s3.Client
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

type S3DirectoryRepository struct {
	sync.Mutex
	connectionRepository connection_deck.Repository
	logger               *log.Logger
	cache                map[connection_deck.ConnectionID]*s3Session
}

var _ directory.Repository = (*S3DirectoryRepository)(nil)

func NewS3DirectoryRepository(
	connectionsRepository connection_deck.Repository,
	bus event.Bus,
	notifier notification.Repository,
) (*S3DirectoryRepository, error) {
	r := &S3DirectoryRepository{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		cache:                make(map[connection_deck.ConnectionID]*s3Session),
	}

	go r.listen(bus, notifier)

	return r, nil
}

func (r *S3DirectoryRepository) GetFileContent(
	ctx context.Context,
	connId connection_deck.ConnectionID,
	file *directory.File,
) (*directory.Content, error) {
	s, err := r.getSession(ctx, connId)
	if err != nil {
		return nil, fmt.Errorf("GetFileContent: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.connection.Bucket()),
		Key:    aws.String(mapFileToKey(file)),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, r.manageAwsSdkError(err, file.FullPath(), s)
	}

	defer result.Body.Close() //nolint:errcheck

	buff := new(bytes.Buffer)
	if _, err = buff.ReadFrom(result.Body); err != nil {
		return nil, fmt.Errorf("fail reading the body content: %w", err)
	}

	rd, w, _ := os.Pipe()
	defer w.Close() //nolint:errcheck
	if _, err := w.Write(buff.Bytes()); err != nil {
		return nil, fmt.Errorf("fail writing the body content: %w", err)
	}

	content := directory.NewFileContent(file, directory.ContentFromFile(rd))

	return content, nil
}

func (r *S3DirectoryRepository) listen(bus event.Bus, notifier notification.Repository) {
	events := bus.Subscribe(
		directory.CreatedEventType,
		directory.DeletedEventType,
		directory.FileCreatedEventType,
		directory.FileDeletedEventType,
		directory.ContentUploadedEventType,
		directory.ContentDownloadEventType,
		directory.LoadEventType,
		directory.FileLoadEventType,
	)

	for {
		select {
		case evt := <-events:
			go func() {
				notifier.NotifyDebug(fmt.Sprintf("[INFRA] received event: %s", evt.Type()))

				ctx := evt.Context()
				if ctx == nil {
					ctx = context.Background()
				}

				switch evt.Type() {
				case directory.CreatedEventType:
					e := evt.(directory.CreatedEvent)
					if err := r.handleDirectoryCreation(ctx, e); err != nil {
						notifier.NotifyError(fmt.Errorf("failed creating directory: %w", err))
						bus.Publish(directory.NewCreatedFailureEvent(err, e.Parent()))
						return
					}
					bus.Publish(directory.NewCreatedSuccessEvent(e.Parent(), e.Directory()))

				case directory.DeletedEventType:
					err := fmt.Errorf("deleting directories is not yet implemented")
					notifier.NotifyError(err)
					bus.Publish(directory.NewDeletedFailureEvent(err))

				case directory.FileCreatedEventType:
					err := fmt.Errorf("file creation is not yet implemented")
					notifier.NotifyError(err)
					bus.Publish(directory.NewFileCreatedFailureEvent(err))

				case directory.FileDeletedEventType:
					e := evt.(directory.FileDeletedEvent)
					if err := r.handleFileDeletion(ctx, e); err != nil {
						notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
						bus.Publish(directory.NewFileDeletedFailureEvent(err, e.Parent()))
						return
					}
					bus.Publish(directory.NewFileDeletedSuccessEvent(e.Parent(), e.File()))

				case directory.ContentUploadedEventType:
					e := evt.(directory.ContentUploadedEvent)
					file, err := r.handleUpload(ctx, e)
					if err != nil {
						notifier.NotifyError(fmt.Errorf("failed uploading file: %w", err))
						bus.Publish(directory.NewContentUploadedFailureEvent(err, e.Directory()))
						return
					}
					bus.Publish(directory.NewContentUploadedSuccessEvent(e.Directory(), file))

				case directory.ContentDownloadEventType:
					e := evt.(directory.ContentDownloadedEvent)
					if err := r.handleDownload(ctx, e); err != nil {
						notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
						bus.Publish(directory.NewContentDownloadedFailureEvent(err))
						return
					}
					bus.Publish(directory.NewContentDownloadedSuccessEvent(e.Content()))

				case directory.LoadEventType:
					notifier.NotifyDebug("[INFRA] loading directory...")
					e := evt.(directory.LoadEvent)
					dir := e.Directory()
					subDirs, files, err := r.handleLoading(ctx, e)
					if err != nil {
						notifier.NotifyError(fmt.Errorf("failed loading directory: %w", err))
						bus.Publish(directory.NewLoadFailureEvent(err, dir))
						return
					}
					notifier.NotifyDebug("[INFRA] loading directory success, publishing...")
					bus.Publish(directory.NewLoadSuccessEvent(dir, subDirs, files))
					notifier.NotifyDebug("[INFRA] loading directory success, published.")

				case directory.FileLoadEventType:
					notifier.NotifyDebug("[INFRA] loading file...")
					e := evt.(directory.FileLoadEvent)
					obj, err := r.handleLoadFile(ctx, e)
					if err != nil {
						notifier.NotifyError(fmt.Errorf("failed loading file: %w", err))
						bus.Publish(directory.NewFileLoadFailureEvent(err, e.File()))
						return
					}
					notifier.NotifyDebug("[INFRA] loading file success, publishing...")
					bus.Publish(directory.NewFileLoadSuccessEvent(e.File(), obj))
					notifier.NotifyDebug("[INFRA] loading file success, published.")
				}

				notifier.NotifyDebug(fmt.Sprintf("[INFRA] event: %s processed", evt.Type()))
			}()
		}
	}
}
