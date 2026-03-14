package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreatedEventType        event.Type = "event.directory.created"
	DeletedEventType        event.Type = "event.directory.deleted"
	LoadEventType           event.Type = "event.directory.load"
	RenameEventType         event.Type = "event.directory.rename"
	RenameResumeEventType   event.Type = "event.directory.rename.resume"
	UserValidationEventType event.Type = "event.directory.user.validation"
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

type withDirContent struct {
	files   []*File
	subDirs []*Directory
}

func (e withDirContent) Files() []*File {
	return e.files
}

func (e withDirContent) SubDirectories() []*Directory {
	return e.subDirs
}

type LoadSuccessEvent struct {
	event.BaseEvent
	withDirectory
	withDirContent
}

func NewLoadSuccessEvent(directory *Directory, subDirs []*Directory, files []*File) LoadSuccessEvent {
	return LoadSuccessEvent{
		event.NewBaseEvent(LoadEventType.AsSuccess()),
		withDirectory{directory},
		withDirContent{
			files,
			subDirs,
		},
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

type withNewName struct {
	newName string
}

func (e withNewName) NewName() string {
	return e.newName
}

type RenameEvent struct {
	event.BaseEvent
	withDirectory
	withNewName
}

func NewRenameEvent(directory *Directory, newName string, opts ...event.Option) RenameEvent {
	return RenameEvent{
		event.NewBaseEvent(RenameEventType, opts...),
		withDirectory{directory},
		withNewName{newName},
	}
}

type RenameSuccessEvent struct {
	event.BaseEvent
	withDirectory
	withNewName
}

func NewRenameSuccessEvent(directory *Directory, newName string, opts ...event.Option) RenameSuccessEvent {
	return RenameSuccessEvent{
		event.NewBaseEvent(RenameEventType.AsSuccess(), opts...),
		withDirectory{directory},
		withNewName{newName},
	}
}

type RenameFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
	withNewName
}

func NewRenameFailureEvent(err error, directory *Directory, newName string) RenameFailureEvent {
	return RenameFailureEvent{
		event.NewBaseErrorEvent(RenameEventType.AsFailure(), err),
		withDirectory{directory},
		withNewName{newName},
	}
}

type withReason struct {
	reason event.Event
}

func (e withReason) Reason() event.Event {
	return e.reason
}

type RenameResumeEvent struct {
	event.BaseEvent
	withDirectory
	dstDir *Directory
}

func NewRenameResumeEvent(srcDir *Directory, dstDir *Directory, opts ...event.Option) RenameResumeEvent {
	return RenameResumeEvent{
		event.NewBaseEvent(RenameResumeEventType, opts...),
		withDirectory{srcDir},
		dstDir,
	}
}

func (e RenameResumeEvent) DstDir() *Directory {
	return e.dstDir
}

type UserValidationEvent struct {
	event.BaseEvent
	withDirectory
	withReason
	message string
}

func NewUserValidationEvent(directory *Directory, reason event.Event, msg string, opts ...event.Option) UserValidationEvent {
	return UserValidationEvent{
		event.NewBaseEvent(UserValidationEventType, opts...),
		withDirectory{directory},
		withReason{reason},
		msg,
	}
}

func (e UserValidationEvent) Message() string {
	return e.message
}

type UserValidationSuccessEvent struct {
	event.BaseEvent
	withDirectory
	withReason
	validated bool
}

func NewUserValidationSuccessEvent(directory *Directory, reason event.Event, validated bool, opts ...event.Option) UserValidationSuccessEvent {
	return UserValidationSuccessEvent{
		event.NewBaseEvent(UserValidationEventType.AsSuccess(), opts...),
		withDirectory{directory},
		withReason{reason},
		validated,
	}
}

func (e UserValidationSuccessEvent) Validated() bool {
	return e.validated
}
