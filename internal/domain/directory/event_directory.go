package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreatedEventType       event.Type = "event.directory.created"
	DeletedEventType       event.Type = "event.directory.deleted"
	LoadEventType          event.Type = "event.directory.load"
	RenameEventType        event.Type = "event.directory.rename"
	RenameRecoverEventType event.Type = "event.directory.rename.recovery"

	UserValidationEventType         event.Type = "event.directory.user.validation"
	UserValidationAcceptedEventType event.Type = "event.directory.user.validation.accepted"
	UserValidationRefusedEventType  event.Type = "event.directory.user.validation.refused"
)

type CreatedEvent struct {
	event.BaseEvent
	ParentDirectory *Directory
	Directory       *Directory
}

// CreatedSuccessEvent represents an event triggered when a directory creation process completes successfully.
// It contains the reference of the new directory created
type CreatedSuccessEvent struct {
	event.BaseEvent
	ParentDirectory *Directory
	Directory       *Directory
}

type CreatedFailureEvent struct {
	event.BaseFailureEvent
	ParentDirectory *Directory
}

func NewCreatedEvent(parent *Directory, newDir *Directory, opts ...event.Option) CreatedEvent {
	return CreatedEvent{
		event.NewBaseEvent(CreatedEventType, opts...),
		parent,
		newDir,
	}
}

func (e CreatedEvent) NewSuccessEvent(opts ...event.Option) CreatedSuccessEvent {
	return CreatedSuccessEvent{
		e.NewBaseSuccess(opts...),
		e.ParentDirectory,
		e.Directory,
	}
}

func (e CreatedEvent) NewFailureEvent(err error) CreatedFailureEvent {
	return CreatedFailureEvent{
		e.NewBaseFailure(err),
		e.ParentDirectory,
	}
}

type DeletedEvent struct {
	event.BaseEvent
	Directory      *Directory
	DeletedDirPath Path
}

type DeletedSuccessEvent struct {
	event.BaseEvent
	Directory *Directory
}

type DeletedFailureEvent struct {
	event.BaseFailureEvent
}

func NewDeletedEvent(directory *Directory, deletedDirPath Path, opts ...event.Option) DeletedEvent {
	return DeletedEvent{
		event.NewBaseEvent(DeletedEventType, opts...),
		directory,
		deletedDirPath,
	}
}

func NewDeletedSuccessEvent(directory *Directory, opts ...event.Option) DeletedSuccessEvent {
	return DeletedSuccessEvent{
		event.NewBaseEvent(DeletedEventType.AsSuccess(), opts...),
		directory,
	}
}

func NewDeletedFailureEvent(err error) DeletedFailureEvent {
	return DeletedFailureEvent{
		event.NewBaseFailureEvent(DeletedEventType.AsFailure(), err),
	}
}

type LoadEvent struct {
	event.BaseEvent
	Directory *Directory
}

func NewLoadEvent(directory *Directory, opts ...event.Option) LoadEvent {
	return LoadEvent{
		event.NewBaseEvent(LoadEventType, opts...),
		directory,
	}
}

type LoadSuccessEvent struct {
	event.BaseEvent
	Directory      *Directory
	Files          []*File
	SubDirectories []*Directory
}

func NewLoadSuccessEvent(directory *Directory, subDirs []*Directory, files []*File) LoadSuccessEvent {
	return LoadSuccessEvent{
		event.NewBaseEvent(LoadEventType.AsSuccess()),
		directory,
		files,
		subDirs,
	}
}

type LoadFailureEvent struct {
	event.BaseFailureEvent
	Directory *Directory
}

func NewLoadFailureEvent(err error, dir *Directory) LoadFailureEvent {
	return LoadFailureEvent{
		event.NewBaseFailureEvent(LoadEventType.AsFailure(), err),
		dir,
	}
}

type RenameEvent struct {
	event.BaseEvent
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

type RenameSuccessEvent struct {
	event.BaseEvent
	Directory *Directory
	NewName   string
}

func NewRenameSuccessEvent(directory *Directory, newName string, opts ...event.Option) RenameSuccessEvent {
	return RenameSuccessEvent{
		event.NewBaseEvent(RenameEventType.AsSuccess(), opts...),
		directory,
		newName,
	}
}

type RenameFailureEvent struct {
	event.BaseFailureEvent
	Directory *Directory
	NewName   string
}

func NewRenameFailureEvent(err error, directory *Directory, newName string) RenameFailureEvent {
	return RenameFailureEvent{
		event.NewBaseFailureEvent(RenameEventType.AsFailure(), err),
		directory,
		newName,
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

type UserValidationEvent struct {
	event.BaseEvent
	Directory *Directory
	Reason    event.Event
	Message   string
}

func NewUserValidationEvent(directory *Directory, reason event.Event, msg string, opts ...event.Option) UserValidationEvent {
	return UserValidationEvent{
		event.NewBaseEvent(UserValidationEventType, opts...),
		directory,
		reason,
		msg,
	}
}

type UserValidationAcceptedEvent struct {
	event.BaseEvent
	Directory *Directory
	Reason    event.Event
}

func NewUserValidationAcceptedEvent(directory *Directory, reason event.Event, opts ...event.Option) UserValidationAcceptedEvent {
	return UserValidationAcceptedEvent{
		event.NewBaseEvent(UserValidationAcceptedEventType, opts...),
		directory,
		reason,
	}
}

type UserValidationRefusedEvent struct {
	event.BaseEvent
	Directory *Directory
	Reason    event.Event
}

func NewUserValidationRefusedEvent(directory *Directory, reason event.Event, opts ...event.Option) UserValidationRefusedEvent {
	return UserValidationRefusedEvent{
		event.NewBaseEvent(UserValidationRefusedEventType, opts...),
		directory,
		reason,
	}
}
