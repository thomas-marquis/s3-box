package directory

import "github.com/thomas-marquis/s3-box/internal/domain/connection_deck"

type DirectoryEvent interface {
	Event
	Directory() *Directory
}

type directoryCreatedEvent struct {
	baseEvent
	directory *Directory
}

var _ DirectoryEvent = &directoryCreatedEvent{}

func newDirectoryCreatedEvent(connectionID connection_deck.ConnectionID, directory *Directory) directoryCreatedEvent {
	return directoryCreatedEvent{baseEvent{connectionID, nil, nil, nil}, directory}
}

func (e directoryCreatedEvent) Name() string {
	return CreatedEventName
}

func (e directoryCreatedEvent) Directory() *Directory {
	return e.directory
}

type directoryDeletedEvent struct {
	baseEvent
	directory *Directory
}

var _ DirectoryEvent = &directoryDeletedEvent{}

func newDirectoryDeletedEvent(connectionID connection_deck.ConnectionID, directory *Directory) directoryDeletedEvent {
	return directoryDeletedEvent{baseEvent{connectionID, nil, nil, nil}, directory}
}

func (e directoryDeletedEvent) Name() string {
	return DeletedEventName
}

func (e directoryDeletedEvent) Directory() *Directory {
	return e.directory
}
