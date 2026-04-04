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
		On(event.Is(directory.CreateTriggeredType), h.handleCreateDirectory).
		On(event.Is(directory.DeleteTriggeredType), h.handleDeleteDirectory).
		On(event.Is(directory.CreateFileTriggeredType), h.handleCreateFile).
		On(event.Is(directory.DeleteFileTriggeredType), h.handleDeleteFile).
		On(event.Is(directory.UploadFileTriggeredType), h.handleUploadFile).
		On(event.Is(directory.DownloadFileTriggeredType), h.handleDownloadFile).
		On(event.Is(directory.LoadTriggeredType), h.handleLoadDirectory).
		On(event.Is(directory.LoadFileTriggeredType), h.handleLoadFile).
		On(event.Is(directory.UserValidationAcceptedType), h.handleRenameDirectory).
		On(event.Is(directory.RenameFileTriggeredType), h.handleRenameFile).
		On(event.Is(directory.RenameTriggeredType), h.handleRenameRequest).
		On(event.Is(directory.RenameRecoveryTriggeredType), h.handleRenameRecovery).
		On(event.IsOneOf(
			connection_deck.RemoveConnectionSucceededType,
			connection_deck.RemoveConnectionFailedType,
		), h.handleConnectionChanged).
		ListenNonBlocking() // TODO: set a limit of simultaneous tasks
}

func (h *EventHandler) handleConnectionChanged(evt event.Event) {
	if pl, ok := evt.Payload.(connection_deck.ConnectionGetter); ok {
		cId := pl.Connection().ID()
		h.clientFactory.Remove(cId)
	}
}
