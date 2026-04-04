package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreateEventType        event.Type = "event.directory.create"
	DeleteEventType        event.Type = "event.directory.delete"
	LoadEventType          event.Type = "event.directory.load"
	RenameEventType        event.Type = "event.directory.rename"
	RenameRecoverEventType event.Type = "event.directory.rename.recovery"

	UserValidationEventType         event.Type = "event.directory.user.validation"
	UserValidationAcceptedEventType event.Type = "event.directory.user.validation.accepted"
	UserValidationRefusedEventType  event.Type = "event.directory.user.validation.refused"
)

type CreateEvent struct {
	event.BaseEvent
	ParentDirectory *Directory
	Directory       *Directory
}

// CreateSuccessEvent represents an event triggered when a directory creation process completes successfully.
// It contains the reference of the new directory created
type CreateSuccessEvent struct {
	event.BaseEvent
	ParentDirectory *Directory
	Directory       *Directory
}

type CreatedFailureEvent struct {
	event.BaseFailureEvent
	ParentDirectory *Directory
}

func NewCreatedEvent(parent *Directory, newDir *Directory, opts ...event.Option) CreateEvent {
	return CreateEvent{
		event.NewBaseEvent(CreateEventType, opts...),
		parent,
		newDir,
	}
}

func (e CreateEvent) NewSuccessEvent(opts ...event.Option) CreateSuccessEvent {
	return CreateSuccessEvent{
		e.NewBaseSuccess(opts...),
		e.ParentDirectory,
		e.Directory,
	}
}

func (e CreateEvent) NewFailureEvent(err error) CreatedFailureEvent {
	return CreatedFailureEvent{
		e.NewBaseFailure(err),
		e.ParentDirectory,
	}
}

func (e CreateSuccessEvent) NewFailureEvent(err error) CreatedFailureEvent {
	return CreatedFailureEvent{
		e.NewBaseFailure(err),
		e.ParentDirectory,
	}
}

type DeleteEvent struct {
	event.BaseEvent
	Directory      *Directory
	DeletedDirPath Path
}

type DeleteSuccessEvent struct {
	event.BaseEvent
	Directory *Directory
}

type DeleteFailureEvent struct {
	event.BaseFailureEvent
}

func NewDeleteEvent(directory *Directory, deletedDirPath Path, opts ...event.Option) DeleteEvent {
	return DeleteEvent{
		event.NewBaseEvent(DeleteEventType, opts...),
		directory,
		deletedDirPath,
	}
}

func (e DeleteEvent) NewSuccessEvent(opts ...event.Option) DeleteSuccessEvent {
	return DeleteSuccessEvent{
		e.NewBaseSuccess(opts...),
		e.Directory,
	}
}

func (e DeleteEvent) NewFailureEvent(err error) DeleteFailureEvent {
	return DeleteFailureEvent{
		e.NewBaseFailure(err),
	}
}

type LoadEvent struct {
	event.BaseEvent
	Directory *Directory
}

type LoadSuccessEvent struct {
	event.BaseEvent
	Directory      *Directory
	Files          []*File
	SubDirectories []*Directory
}

type LoadFailureEvent struct {
	event.BaseFailureEvent
	Directory *Directory
}

func NewLoadEvent(directory *Directory, opts ...event.Option) LoadEvent {
	return LoadEvent{
		event.NewBaseEvent(LoadEventType, opts...),
		directory,
	}
}

func NewLoadSuccessEvent(directory *Directory, subDirs []*Directory, files []*File, opts ...event.Option) LoadSuccessEvent {
	return LoadSuccessEvent{
		event.NewBaseEvent(LoadEventType.AsSuccess(), opts...),
		directory,
		files,
		subDirs,
	}
}

func NewLoadFailureEvent(err error, directory *Directory) LoadFailureEvent {
	return LoadFailureEvent{
		event.NewBaseFailureEvent(LoadEventType.AsFailure(), err),
		directory,
	}
}

func (e LoadEvent) NewSuccessEvent(directory *Directory, subDirs []*Directory, files []*File, opts ...event.Option) LoadSuccessEvent {
	//return LoadSuccessEvent{
	//	e.NewBaseSuccess(opts...),
	//	directory,
	//	files,
	//	subDirs,
	//}

	opts = append([]event.Option{event.WithRef(e.Ref())}, opts...)
	return NewLoadSuccessEvent(directory, subDirs, files, opts...)
}

func (e LoadEvent) NewFailureEvent(err error) LoadFailureEvent {
	return LoadFailureEvent{
		e.NewBaseFailure(err),
		e.Directory,
	}
}

type RenameEvent struct {
	event.BaseEvent
	Directory *Directory
	NewName   string
}

type RenameSuccessEvent struct {
	event.BaseEvent
	Directory *Directory
	NewName   string
}

type RenameFailureEvent struct {
	event.BaseFailureEvent
	Directory *Directory
	NewName   string
}

func NewRenameEvent(directory *Directory, newName string, opts ...event.Option) RenameEvent {
	return RenameEvent{
		event.NewBaseEvent(RenameEventType, opts...),
		directory,
		newName,
	}
}

func (e RenameEvent) NewSuccessEvent(opts ...event.Option) RenameSuccessEvent {
	return RenameSuccessEvent{
		e.NewBaseSuccess(opts...),
		e.Directory,
		e.NewName,
	}
}

func (e RenameEvent) NewFailureEvent(err error) RenameFailureEvent {
	return RenameFailureEvent{
		e.NewBaseFailure(err),
		e.Directory,
		e.NewName,
	}
}

type RenameRecoverEvent struct {
	event.BaseEvent
	Directory *Directory
	DstDir    *Directory
	Choice    RecoveryChoice
}

func NewRenameRecoverEvent(srcDir *Directory, dstDir *Directory, choice RecoveryChoice, opts ...event.Option) RenameRecoverEvent {
	return RenameRecoverEvent{
		event.NewBaseEvent(RenameRecoverEventType, opts...),
		srcDir,
		dstDir,
		choice,
	}
}

func (e RenameRecoverEvent) InvertDirs() RenameRecoverEvent {
	return RenameRecoverEvent{
		event.NewBaseEvent(RenameRecoverEventType,
			event.WithContext(e.Context()), event.WithRef(e.Ref())),
		e.DstDir,
		e.Directory,
		e.Choice,
	}
}

func (e RenameRecoverEvent) NewRenameFailureEvent(err error) RenameFailureEvent {
	return RenameFailureEvent{
		event.NewBaseFailureEvent(RenameEventType.AsFailure(), err,
			event.WithContext(e.Context()), event.WithRef(e.Ref())),
		e.Directory,
		e.DstDir.Name(),
	}
}

func (e RenameRecoverEvent) NewRenameSuccessEvent() RenameSuccessEvent {
	return RenameSuccessEvent{
		event.NewBaseEvent(RenameEventType.AsSuccess(),
			event.WithContext(e.Context()), event.WithRef(e.Ref())),
		e.Directory,
		e.DstDir.Name(),
	}
}

type UserValidationEvent struct {
	event.BaseEvent
	Directory *Directory
	Reason    event.Event
	Message   string
}

type UserValidationAcceptedEvent struct {
	event.BaseEvent
	Directory *Directory
	Reason    event.Event
}

type UserValidationRefusedEvent struct {
	event.BaseEvent
	Directory *Directory
	Reason    event.Event
}

func NewUserValidationEvent(directory *Directory, reason event.Event, msg string, opts ...event.Option) UserValidationEvent {
	return UserValidationEvent{
		event.NewBaseEvent(UserValidationEventType, opts...),
		directory,
		reason,
		msg,
	}
}

func (e UserValidationEvent) NewAcceptedEvent() UserValidationAcceptedEvent {
	return UserValidationAcceptedEvent{
		event.NewBaseEvent(UserValidationAcceptedEventType,
			event.WithContext(e.Context()), event.WithRef(e.Ref())),
		e.Directory,
		e.Reason,
	}
}

func (e UserValidationEvent) NewRefusedEvent() UserValidationRefusedEvent {
	return UserValidationRefusedEvent{
		event.NewBaseEvent(UserValidationRefusedEventType,
			event.WithContext(e.Context()), event.WithRef(e.Ref())),
		e.Directory,
		e.Reason,
	}
}
