package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	FileCreatedEventType event.Type = "event.file.created"
	FileDeletedEventType event.Type = "event.file.deleted"
)

type withFile struct {
	file *File
}

func (e withFile) File() *File {
	return e.file
}

type withConnectionID struct {
	connectionID connection_deck.ConnectionID
}

func (e withConnectionID) ConnectionID() connection_deck.ConnectionID {
	return e.connectionID
}

type FileCreatedEvent struct {
	event.BaseEvent
	withFile
	withConnectionID
}

func NewFileCreatedEvent(connectionID connection_deck.ConnectionID, file *File, opts ...event.Option) FileCreatedEvent {
	return FileCreatedEvent{
		event.NewBaseEvent(FileCreatedEventType, opts...),
		withFile{file},
		withConnectionID{connectionID},
	}
}

type FileCreatedSuccessEvent struct {
	event.BaseEvent
	withFile
}

func NewFileCreatedSuccessEvent(file *File, opts ...event.Option) FileCreatedSuccessEvent {
	return FileCreatedSuccessEvent{
		event.NewBaseEvent(FileCreatedEventType.AsSuccess(), opts...),
		withFile{file},
	}
}

type FileCreatedFailureEvent struct {
	event.BaseErrorEvent
}

func NewFileCreatedFailureEvent(err error) FileCreatedFailureEvent {
	return FileCreatedFailureEvent{
		event.NewBaseErrorEvent(FileCreatedEventType.AsFailure(), err),
	}
}

type FileDeletedEvent struct {
	event.BaseEvent
	withFile
	withConnectionID
}

func NewFileDeletedEvent(connectionID connection_deck.ConnectionID, file *File, opts ...event.Option) FileDeletedEvent {
	return FileDeletedEvent{
		event.NewBaseEvent(FileDeletedEventType, opts...),
		withFile{file},
		withConnectionID{connectionID},
	}
}

type FileDeletedSuccessEvent struct {
	event.BaseEvent
	withFile
}

func NewFileDeletedSuccessEvent(file *File, opts ...event.Option) FileDeletedSuccessEvent {
	return FileDeletedSuccessEvent{
		event.NewBaseEvent(FileDeletedEventType.AsSuccess(), opts...),
		withFile{file},
	}
}

type FileDeletedFailureEvent struct {
	event.BaseErrorEvent
}

func NewFileDeletedFailureEvent(err error) FileDeletedFailureEvent {
	return FileDeletedFailureEvent{
		event.NewBaseErrorEvent(FileDeletedEventType.AsFailure(), err),
	}
}
