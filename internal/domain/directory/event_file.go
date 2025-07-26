package directory

import "github.com/thomas-marquis/s3-box/internal/domain/connection_deck"

type FileEvent interface {
	Event
	File() *File
}

type fileCreatedEvent struct {
	baseEvent
	file *File
}

func newFileCreatedEvent(connectionID connection_deck.ConnectionID, file *File) fileCreatedEvent {
	return fileCreatedEvent{baseEvent{connectionID, nil, nil, nil}, file}
}

func (e fileCreatedEvent) Name() string {
	return FileCreatedEventName
}

func (e fileCreatedEvent) File() *File {
	return e.file
}

type fileDeletedEvent struct {
	baseEvent
	file *File
}

func newFileDeletedEvent(connectionID connection_deck.ConnectionID, file *File) fileDeletedEvent {
	return fileDeletedEvent{baseEvent{connectionID, nil, nil, nil}, file}
}

func (e fileDeletedEvent) Name() string {
	return FileDeletedEventName
}

func (e fileDeletedEvent) File() *File {
	return e.file
}
