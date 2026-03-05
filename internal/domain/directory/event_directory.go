package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreatedEventType event.Type = "event.directory.created"
	DeletedEventType event.Type = "event.directory.deleted"
	LoadEventType    event.Type = "event.directory.load"
	RenamedEventType event.Type = "event.directory.renamed"
	UserValidation   event.Type = "event.directory.user.validation"
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
	files   []*File
	subDirs []*Directory
}

func NewLoadSuccessEvent(directory *Directory, subDirs []*Directory, files []*File) LoadSuccessEvent {
	return LoadSuccessEvent{
		event.NewBaseEvent(LoadEventType.AsSuccess()),
		withDirectory{directory},
		files,
		subDirs,
	}
}

func (e *LoadSuccessEvent) Files() []*File {
	return e.files
}

func (e *LoadSuccessEvent) SubDirectories() []*Directory {
	return e.subDirs
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

type withOldPath struct {
	oldPath Path
}

func (e withOldPath) OldPath() Path {
	return e.oldPath
}

type withNewName struct {
	newName string
}

func (e withNewName) NewName() string {
	return e.newName
}

type RenamedEvent struct {
	event.BaseEvent
	withDirectory
	withOldPath // TODO: is it necessary??
	withNewName
}

func NewRenamedEvent(directory *Directory, oldPath Path, newName string, opts ...event.Option) RenamedEvent {
	return RenamedEvent{
		event.NewBaseEvent(RenamedEventType, opts...),
		withDirectory{directory},
		withOldPath{oldPath}, // TODO: is it necessary??
		withNewName{newName},
	}
}

type RenamedSuccessEvent struct {
	event.BaseEvent
	withDirectory
	withOldPath
	withNewName
}

func NewRenamedSuccessEvent(directory *Directory, oldPath Path, newName string, opts ...event.Option) RenamedSuccessEvent {
	return RenamedSuccessEvent{
		event.NewBaseEvent(RenamedEventType.AsSuccess(), opts...),
		withDirectory{directory},
		withOldPath{oldPath},
		withNewName{newName},
	}
}

type RenamedFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
	withOldPath
}

func NewRenamedFailureEvent(err error, directory *Directory, oldPath Path) RenamedFailureEvent {
	return RenamedFailureEvent{
		event.NewBaseErrorEvent(RenamedEventType.AsFailure(), err),
		withDirectory{directory},
		withOldPath{oldPath},
	}
}

type withValidationReason struct {
	reason string
}

func (e withValidationReason) Reason() string {
	return e.reason
}

type UserValidationEvent struct {
	event.BaseEvent
	withDirectory
	withValidationReason
	message string
}

func NewUserValidationEvent(directory *Directory, reason string, msg string, opts ...event.Option) UserValidationEvent {
	return UserValidationEvent{
		event.NewBaseEvent(UserValidation, opts...),
		withDirectory{directory},
		withValidationReason{reason},
		msg,
	}
}

func (e UserValidationEvent) Message() string {
	return e.message
}

type UserValidationSuccessEvent struct {
	event.BaseEvent
	withDirectory
	withValidationReason
	validated bool
}

func NewUserValidationSuccessEvent(directory *Directory, reason string, validated bool, opts ...event.Option) UserValidationSuccessEvent {
	return UserValidationSuccessEvent{
		event.NewBaseEvent(UserValidation.AsSuccess(), opts...),
		withDirectory{directory},
		withValidationReason{reason},
		validated,
	}
}

func (e UserValidationSuccessEvent) Validated() bool {
	return e.validated
}

type UserValidationFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
	withValidationReason
}

func NewUserValidationFailureEvent(err error, directory *Directory, reason string) UserValidationFailureEvent {
	return UserValidationFailureEvent{
		event.NewBaseErrorEvent(UserValidation.AsFailure(), err),
		withDirectory{directory},
		withValidationReason{reason},
	}
}
