package s3

import (
	"log"
	"os"
	"sync"

	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type EventHandler struct {
	sync.Mutex
	connectionRepository connection_deck.Repository
	logger               *log.Logger
	bus                  event.Bus
	notifier             notification.Repository
	s3ClientOptions      []func(*s3.Options)

	clientFactory s3client.Factory
}

func NewS3EventHandler(
	connectionsRepository connection_deck.Repository,
	bus event.Bus,
	notifier notification.Repository,
	s3ClientOptions ...func(*s3.Options),
) *EventHandler {
	return &EventHandler{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		bus:                  bus,
		notifier:             notifier,
		s3ClientOptions:      s3ClientOptions,
		clientFactory:        s3client.NewFactory(connectionsRepository, notifier, s3ClientOptions...),
	}
}

func (r *EventHandler) Listen() {
	r.bus.Subscribe().
		On(event.Is(directory.CreatedEventType), r.handleCreateDirectory).
		On(event.Is(directory.DeletedEventType), r.handleDeleteDirectory).
		On(event.Is(directory.FileCreatedEventType), r.handleCreateFile).
		On(event.Is(directory.FileDeletedEventType), r.handleDeleteFile).
		On(event.Is(directory.FileUploadEventType), r.handleUploadFile).
		On(event.Is(directory.FileDownloadEventType), r.handleDownloadFile).
		On(event.Is(directory.LoadEventType), r.handleLoadDirectory).
		On(event.Is(directory.FileLoadEventType), r.handleLoadFile).
		On(event.Is(directory.UserValidationAcceptedEventType), r.handleRenameDirectory).
		On(event.Is(directory.FileRenameEventType), r.handleRenameFile).
		On(event.Is(directory.RenameEventType), r.handleRenameRequest).
		On(event.Is(directory.RenameRecoverEventType), r.handleRenameRecovery).
		On(event.IsOneOf(
			connection_deck.RemoveEventType.AsSuccess(),
			connection_deck.UpdateEventType.AsSuccess(),
		), r.handleConnectionChanged).
		ListenNonBlocking() // TODO: set a limit of simultaneous tasks
}

func (r *EventHandler) handleConnectionChanged(evt event.Event) {
	if e, ok := evt.(connection_deck.ConnectionEvent); ok {
		cId := e.Connection().ID()
		r.clientFactory.Remove(cId)
	}
}
