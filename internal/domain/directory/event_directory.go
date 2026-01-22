package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreatedEventType event.Type = "event.directory.created"
	DeletedEventType event.Type = "event.directory.deleted"
	LoadEventType    event.Type = "event.directory.load"
)

type withDirectory struct {
	directory *Directory
}

func (e withDirectory) Directory() *Directory {
	return e.directory
}

type withParentDirectory struct {
	parent *Directory
}

func (e withParentDirectory) Parent() *Directory {
	return e.parent
}

type CreatedEvent struct {
	event.BaseEvent
	withParentDirectory
	withDirectory
}

func NewCreatedEvent(parent *Directory, newDir *Directory, opts ...event.Option) CreatedEvent {
	return CreatedEvent{
		event.NewBaseEvent(CreatedEventType, opts...),
		withParentDirectory{parent},
		withDirectory{newDir},
	}
}

// CreatedSuccessEvent represents an event triggered when a directory creation process completes successfully.
// It contains the reference of the new directory created
type CreatedSuccessEvent struct {
	event.BaseEvent
	withParentDirectory
	withDirectory
}

func NewCreatedSuccessEvent(parent *Directory, newDire *Directory, opts ...event.Option) CreatedSuccessEvent {
	return CreatedSuccessEvent{
		event.NewBaseEvent(CreatedEventType.AsSuccess(), opts...),
		withParentDirectory{parent},
		withDirectory{newDire},
	}
}

type CreatedFailureEvent struct {
	event.BaseErrorEvent
	withParentDirectory
}

func NewCreatedFailureEvent(err error, parent *Directory) CreatedFailureEvent {
	return CreatedFailureEvent{
		event.NewBaseErrorEvent(CreatedEventType.AsFailure(), err),
		withParentDirectory{parent},
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

type LoadEvent struct {
	event.BaseEvent
	withDirectory
}

func NewLoadEvent(directory *Directory, opts ...event.Option) LoadEvent {
	return LoadEvent{
		event.NewBaseEvent(LoadEventType, opts...),
		withDirectory{directory},
	}
}

type LoadSuccessEvent struct {
	event.BaseEvent
	withDirectory
	files       []*File
	subDirPaths []Path
}

func NewLoadSuccessEvent(directory *Directory, subDirPaths []Path, files []*File) LoadSuccessEvent {
	return LoadSuccessEvent{
		event.NewBaseEvent(LoadEventType.AsSuccess()),
		withDirectory{directory},
		files,
		subDirPaths,
	}
}

type LoadFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
}

func NewLoadFailureEvent(err error, dir *Directory) LoadFailureEvent {
	return LoadFailureEvent{
		event.NewBaseErrorEvent(LoadEventType.AsFailure(), err),
		withDirectory{dir},
	}
}
