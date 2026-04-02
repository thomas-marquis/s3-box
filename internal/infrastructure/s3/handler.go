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

func (h *EventHandler) Listen() {
	h.bus.Subscribe().
		On(event.Is(directory.CreatedEventType), h.handleCreateDirectory).
		On(event.Is(directory.DeletedEventType), h.handleDeleteDirectory).
		On(event.Is(directory.FileCreatedEventType), h.handleCreateFile).
		On(event.Is(directory.FileDeletedEventType), h.handleDeleteFile).
		On(event.Is(directory.FileUploadEventType), h.handleUploadFile).
		On(event.Is(directory.FileDownloadEventType), h.handleDownloadFile).
		On(event.Is(directory.LoadEventType), h.handleLoadDirectory).
		On(event.Is(directory.FileLoadEventType), h.handleLoadFile).
		On(event.Is(directory.UserValidationAcceptedEventType), h.handleRenameDirectory).
		On(event.Is(directory.FileRenameEventType), h.handleRenameFile).
		On(event.Is(directory.RenameEventType), h.handleRenameRequest).
		On(event.Is(directory.RenameRecoverEventType), h.handleRenameRecovery).
		On(event.IsOneOf(
			connection_deck.RemoveEventType.AsSuccess(),
			connection_deck.UpdateEventType.AsSuccess(),
		), h.handleConnectionChanged).
		ListenNonBlocking() // TODO: set a limit of simultaneous tasks
}

func (h *EventHandler) handleConnectionChanged(evt event.Event) {
	if e, ok := evt.(connection_deck.ConnectionEvent); ok {
		cId := e.Connection().ID()
		h.clientFactory.Remove(cId)
	}
}
