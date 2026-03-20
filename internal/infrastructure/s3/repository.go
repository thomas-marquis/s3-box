package s3

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
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type s3Session struct {
	connection *connection_deck.Connection
	client     *s3.Client
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

type RepositoryImpl struct {
	sync.Mutex
	connectionRepository connection_deck.Repository
	logger               *log.Logger
	cache                map[connection_deck.ConnectionID]*s3Session
	bus                  event.Bus
	notifier             notification.Repository
	s3ClientOptions      []func(*s3.Options)

	clientFactory s3client.Factory
}

var _ directory.Repository = (*RepositoryImpl)(nil)

func NewRepositoryImpl(
	connectionsRepository connection_deck.Repository,
	bus event.Bus,
	notifier notification.Repository,
	s3ClientOptions ...func(*s3.Options),
) (*RepositoryImpl, error) {
	r := &RepositoryImpl{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		cache:                make(map[connection_deck.ConnectionID]*s3Session),
		bus:                  bus,
		notifier:             notifier,
		s3ClientOptions:      s3ClientOptions,
		clientFactory:        s3client.NewFactory(connectionsRepository, notifier),
	}

	bus.Subscribe().
		On(event.Is(directory.CreatedEventType), r.handleCreateDirectory).
		On(event.Is(directory.DeletedEventType), r.handleDeleteDirectory).
		On(event.Is(directory.FileCreatedEventType), r.handleCreateFile).
		On(event.Is(directory.FileDeletedEventType), r.handleDeleteFile).
		On(event.Is(directory.ContentUploadedEventType), r.handleUploadFile).
		On(event.Is(directory.ContentDownloadEventType), r.handleDownloadFile).
		On(event.Is(directory.LoadEventType), r.handleLoadDirectory).
		On(event.Is(directory.FileLoadEventType), r.handleLoadFile).
		On(event.Is(directory.UserValidationEventType.AsSuccess()), r.handleRenameDirectory).
		On(event.Is(directory.FileRenamedEventType), r.handleRenameFile).
		On(event.Is(directory.RenameEventType), r.handleRenameRequest).
		On(event.Is(directory.RenameRecoverEventType), r.handleRenameRecovery).
		On(event.IsOneOf(
			connection_deck.RemoveEventType.AsSuccess(),
			connection_deck.UpdateEventType.AsSuccess(),
		), r.handleConnectionChanged).
		ListenNonBlocking() // TODO: set a limit of simultaneous tasks

	return r, nil
}

func (r *RepositoryImpl) handleConnectionChanged(evt event.Event) {
	if e, ok := evt.(connection_deck.ConnectionEvent); ok {
		cId := e.Connection().ID()
		r.clientFactory.Remove(cId)
	}
}

func (r *RepositoryImpl) GetFileContent(
	ctx context.Context,
	connId connection_deck.ConnectionID,
	file *directory.File,
) (*directory.Content, error) {
	client, err := r.clientFactory.Get(ctx, connId)
	if err != nil {
		return nil, fmt.Errorf("GetFileContent: %w", err)
	}

	result, err := client.GetObject(ctx, mapFileToKey(file))
	if err != nil {
		return nil, err
	}

	defer result.Body.Close() //nolint:errcheck

	buff := new(bytes.Buffer)
	if _, err = buff.ReadFrom(result.Body); err != nil {
		return nil, fmt.Errorf("fail reading the body content: %w", err)
	}

	rd, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer w.Close() //nolint:errcheck
	if _, err := w.Write(buff.Bytes()); err != nil {
		return nil, fmt.Errorf("fail writing the body content: %w", err)
	}

	content := directory.NewFileContent(file, directory.ContentFromFile(rd))

	return content, nil
}
