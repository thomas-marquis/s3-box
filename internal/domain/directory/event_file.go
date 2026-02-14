package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	FileCreatedEventType event.Type = "event.file.created"
	FileDeletedEventType event.Type = "event.file.deleted"
	FileLoadEventType    event.Type = "event.file.load"
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
	withParentDirectory
}

func NewFileDeletedEvent(connectionID connection_deck.ConnectionID, parent *Directory, file *File, opts ...event.Option) FileDeletedEvent {
	return FileDeletedEvent{
		event.NewBaseEvent(FileDeletedEventType, opts...),
		withFile{file},
		withConnectionID{connectionID},
		withParentDirectory{parent},
	}
}

type FileDeletedSuccessEvent struct {
	event.BaseEvent
	withFile
	withParentDirectory
}

func NewFileDeletedSuccessEvent(parent *Directory, file *File, opts ...event.Option) FileDeletedSuccessEvent {
	return FileDeletedSuccessEvent{
		event.NewBaseEvent(FileDeletedEventType.AsSuccess(), opts...),
		withFile{file},
		withParentDirectory{parent},
	}
}

type FileDeletedFailureEvent struct {
	event.BaseErrorEvent
	withParentDirectory
}

func NewFileDeletedFailureEvent(err error, parent *Directory) FileDeletedFailureEvent {
	return FileDeletedFailureEvent{
		event.NewBaseErrorEvent(FileDeletedEventType.AsFailure(), err),
		withParentDirectory{parent},
	}
}

type FileLoadEvent struct {
	event.BaseEvent
	withFile
	withConnectionID
}

func NewFileLoadEvent(connectionID connection_deck.ConnectionID, file *File, opts ...event.Option) FileLoadEvent {
	return FileLoadEvent{
		event.NewBaseEvent(FileLoadEventType, opts...),
		withFile{file},
		withConnectionID{connectionID},
	}
}

type FileLoadSuccessEvent struct {
	event.BaseEvent
	withFile
	Content FileObject
}

func NewFileLoadSuccessEvent(file *File, content FileObject) FileLoadSuccessEvent {
	return FileLoadSuccessEvent{
		event.NewBaseEvent(FileLoadEventType.AsSuccess()),
		withFile{file},
		content,
	}
}

type FileLoadFailureEvent struct {
	event.BaseErrorEvent
	withFile
}

func NewFileLoadFailureEvent(err error, file *File) FileLoadFailureEvent {
	return FileLoadFailureEvent{
		event.NewBaseErrorEvent(FileLoadEventType.AsFailure(), err),
		withFile{file},
	}
}
