package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreateFileTriggeredEventType = "event.file.create"
	CreateFileSucceededEventType = "event.file.create.succeeded"
	CreateFileFailedEventType    = "event.file.create.failed"

	FileUploadEventType   event.Type = "event.file.upload"
	FileDownloadEventType event.Type = "event.file.download"
)

type CreateFileTriggered struct {
	event.BaseEvent
	File         *File
	ConnectionID connection_deck.ConnectionID
	Directory    *Directory
}

func (e CreateFileTriggered) Type() event.Type {
	return CreateFileTriggeredEventType
}

type CreateFileSucceeded struct {
	event.BaseEvent
	File      *File
	Directory *Directory
}

func (e CreateFileSucceeded) Type() event.Type {
	return CreateFileSucceededEventType
}

type CreateFileFailed struct {
	event.BaseFailureEvent
	Directory *Directory
}

func (e CreateFileFailed) Type() event.Type {
	return CreateFileFailedEventType
}

func NewFileCreatedEvent(connectionID connection_deck.ConnectionID, dir *Directory, file *File, opts ...event.Option) CreateFileTriggered {
	return CreateFileTriggered{
		event.NewBaseEvent(opts...),
		file,
		connectionID,
		dir,
	}
}

func (e CreateFileTriggered) NewSuccessEvent(opts ...event.Option) CreateFileSucceeded {
	return CreateFileSucceeded{
		e.NewBaseSuccess(opts...),
		e.File,
		e.Directory,
	}
}

func (e CreateFileTriggered) NewFailureEvent(err error) CreateFileFailed {
	return CreateFileFailed{
		e.NewBaseFailure(err),
		e.Directory,
	}
}

func (e CreateFileSucceeded) NewFailureEvent(err error) CreateFileFailed {
	return CreateFileFailed{
		e.NewBaseFailure(err),
		e.Directory,
	}
}

const (
	DeleteFileTriggeredEventType event.Type = "event.file.delete"
	DeleteFileSucceededEventType event.Type = "event.file.delete.succeeded"
	DeleteFileFailedEventType    event.Type = "event.file.delete.failed"
)

type DeleteFileTriggered struct {
	event.BaseEvent
	File            *File
	ConnectionID    connection_deck.ConnectionID
	ParentDirectory *Directory
}

func (e DeleteFileTriggered) Type() event.Type {
	return DeleteFileTriggeredEventType
}

type DeleteFileSucceeded struct {
	event.BaseEvent
	File            *File
	ParentDirectory *Directory
}

func (e DeleteFileSucceeded) Type() event.Type {
	return DeleteFileSucceededEventType
}

type DeleteFileFailed struct {
	event.BaseFailureEvent
	ParentDirectory *Directory
}

func (e DeleteFileFailed) Type() event.Type {
	return DeleteFileFailedEventType
}

func NewFileDeleteEvent(connectionID connection_deck.ConnectionID, parent *Directory, file *File, opts ...event.Option) DeleteFileTriggered {
	return DeleteFileTriggered{
		event.NewBaseEvent(opts...),
		file,
		connectionID,
		parent,
	}
}

func (e DeleteFileTriggered) NewSuccessEvent(file *File, opts ...event.Option) DeleteFileSucceeded {
	return DeleteFileSucceeded{
		e.NewBaseSuccess(opts...),
		file,
		e.ParentDirectory,
	}
}

func (e DeleteFileTriggered) NewFailureEvent(err error) DeleteFileFailed {
	return DeleteFileFailed{
		e.NewBaseFailure(err),
		e.ParentDirectory,
	}
}

func (e DeleteFileSucceeded) NewFailureEvent(err error) DeleteFileFailed {
	return DeleteFileFailed{
		e.NewBaseFailure(err),
		e.ParentDirectory,
	}
}

const (
	LoadFileTriggeredEventType event.Type = "event.file.load"
	LoadFileSucceededEventType event.Type = "event.file.load.succeeded"
	LoadFileFailedEventType    event.Type = "event.file.load.failed"
)

type LoadFileTriggered struct {
	event.BaseEvent
	File         *File
	ConnectionID connection_deck.ConnectionID
}

func (e LoadFileTriggered) Type() event.Type {
	return LoadFileTriggeredEventType
}

func NewFileLoadEvent(connectionID connection_deck.ConnectionID, file *File, opts ...event.Option) LoadFileTriggered {
	return LoadFileTriggered{
		event.NewBaseEvent(opts...),
		file,
		connectionID,
	}
}

type LoadFileSucceeded struct {
	event.BaseEvent
	File    *File
	Content FileContent
}

func (e LoadFileSucceeded) Type() event.Type {
	return LoadFileSucceededEventType
}

func NewFileLoadSuccessEvent(file *File, content FileContent) LoadFileSucceeded {
	return LoadFileSucceeded{
		event.NewBaseEvent(),
		file,
		content,
	}
}

type LoadFileFailed struct {
	event.BaseFailureEvent
	File *File
}

func (e LoadFileFailed) Type() event.Type {
	return LoadFileFailedEventType
}

func NewFileLoadFailureEvent(err error, file *File) LoadFileFailed {
	return LoadFileFailed{
		event.NewBaseFailureEvent(LoadFileFailedEventType, err),
		file,
	}
}

const (
	FileRenameEventType event.Type = "event.file.rename"
)

type FileRenameEvent struct {
	event.BaseEvent
	File      *File
	NewName   string
	Directory *Directory
}

func NewFileRenameEvent(dir *Directory, file *File, newName string, opts ...event.Option) FileRenameEvent {
	return FileRenameEvent{
		event.NewBaseEvent(FileRenameEventType, opts...),
		file,
		newName,
		dir,
	}
}

type FileRenameSuccessEvent struct {
	event.BaseEvent
	File      *File
	NewName   string
	Directory *Directory
}

func NewFileRenameSuccessEvent(dir *Directory, file *File, newName string, opts ...event.Option) FileRenameSuccessEvent {
	return FileRenameSuccessEvent{
		event.NewBaseEvent(FileRenameEventType.AsSuccess(), opts...),
		file,
		newName,
		dir,
	}
}

type FileRenameFailureEvent struct {
	event.BaseFailureEvent
	File      *File
	NewName   string
	Directory *Directory
}

func NewFileRenameFailureEvent(err error, dir *Directory, file *File, newName string) FileRenameFailureEvent {
	return FileRenameFailureEvent{
		event.NewBaseFailureEvent(FileRenameEventType.AsFailure(), err),
		file,
		newName,
		dir,
	}
}

type FileUploadEvent struct {
	event.BaseEvent
	Directory *Directory
	SrcPath   string
}

func NewFileUploadEvent(directory *Directory, localFilePath string, opts ...event.Option) FileUploadEvent {
	return FileUploadEvent{
		event.NewBaseEvent(FileUploadEventType, opts...),
		directory,
		localFilePath,
	}
}

type FileUploadSuccessEvent struct {
	event.BaseEvent
	File      *File
	Directory *Directory
}

func NewFileUploadSuccessEvent(directory *Directory, file *File, opts ...event.Option) FileUploadSuccessEvent {
	return FileUploadSuccessEvent{
		event.NewBaseEvent(FileUploadEventType.AsSuccess(), opts...),
		file,
		directory,
	}
}

type FileUploadFailureEvent struct {
	event.BaseFailureEvent
	Directory *Directory
}

func NewFileUploadFailureEvent(err error, dir *Directory) FileUploadFailureEvent {
	return FileUploadFailureEvent{
		event.NewBaseFailureEvent(FileUploadEventType.AsFailure(), err),
		dir,
	}
}

type FileDownloadEvent struct {
	event.BaseEvent
	ConnectionID connection_deck.ConnectionID
	DstPath      string
	File         *File
}

func NewFileDownloadEvent(connectionID connection_deck.ConnectionID, file *File, dstLocalPtah string, opts ...event.Option) FileDownloadEvent {
	return FileDownloadEvent{
		event.NewBaseEvent(FileDownloadEventType, opts...),
		connectionID,
		dstLocalPtah,
		file,
	}
}

type FileDownloadSuccessEvent struct {
	event.BaseEvent
	File *File
}

func NewFileDownloadSuccessEvent(file *File, opts ...event.Option) FileDownloadSuccessEvent {
	return FileDownloadSuccessEvent{
		event.NewBaseEvent(FileDownloadEventType.AsSuccess(), opts...),
		file,
	}
}

type FileDownloadFailureEvent struct {
	event.BaseFailureEvent
}

func NewFileDownloadFailureEvent(err error) FileDownloadFailureEvent {
	return FileDownloadFailureEvent{
		event.NewBaseFailureEvent(FileDownloadEventType.AsFailure(), err),
	}
}
