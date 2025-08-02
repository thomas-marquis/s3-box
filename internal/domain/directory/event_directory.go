package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreatedEventType event.Type = "event.directory.created"
	DeletedEventType event.Type = "event.directory.deleted"
)

type withDirectory struct {
	directory *Directory
}

func (e withDirectory) Directory() *Directory {
	return e.directory
}

type CreatedEvent struct {
	event.BaseEvent
	withDirectory
}

func NewCreatedEvent(directory *Directory, opts ...event.Option) CreatedEvent {
	return CreatedEvent{
		event.NewBaseEvent(CreatedEventType, opts...),
		withDirectory{directory},
	}
}

type CreatedSuccessEvent struct {
	event.BaseEvent
	withDirectory
}

func NewCreatedSuccessEvent(directory *Directory, opts ...event.Option) CreatedSuccessEvent {
	return CreatedSuccessEvent{
		event.NewBaseEvent(CreatedEventType.AsSuccess(), opts...),
		withDirectory{directory},
	}
}

type CreatedFailureEvent struct {
	event.BaseErrorEvent
}

func NewCreatedFailureEvent(err error) CreatedFailureEvent {
	return CreatedFailureEvent{
		event.NewBaseErrorEvent(CreatedEventType.AsFailure(), err),
	}
}

type DeletedEvent struct {
	event.BaseEvent
	withDirectory
	deletedDirPath Path
}

func NewDeletedEvent(directory *Directory, deletedDirPath Path, opts ...event.Option) DeletedEvent {
	return DeletedEvent{
		event.NewBaseEvent(DeletedEventType, opts...),
		withDirectory{directory},
		deletedDirPath,
	}
}

func (e DeletedEvent) DeletedDirPath() Path {
	return e.deletedDirPath
}

type DeletedSuccessEvent struct {
	event.BaseEvent
	withDirectory
}

func NewDeletedSuccessEvent(directory *Directory, opts ...event.Option) DeletedSuccessEvent {
	return DeletedSuccessEvent{
		event.NewBaseEvent(DeletedEventType.AsSuccess(), opts...),
		withDirectory{directory},
	}
}

type DeletedFailureEvent struct {
	event.BaseErrorEvent
}

func NewDeletedFailureEvent(err error) DeletedFailureEvent {
	return DeletedFailureEvent{
		event.NewBaseErrorEvent(DeletedEventType.AsFailure(), err),
	}
}
