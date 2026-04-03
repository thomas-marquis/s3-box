package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	FileCreatedEventType  event.Type = "event.file.created"
	FileDeletedEventType  event.Type = "event.file.deleted"
	FileLoadEventType     event.Type = "event.file.load"
	FileRenameEventType   event.Type = "event.file.rename"
	FileUploadEventType   event.Type = "event.file.upload"
	FileDownloadEventType event.Type = "event.file.download"
)

type FileCreatedEvent struct {
	event.BaseEvent
	File         *File
	ConnectionID connection_deck.ConnectionID
	Directory    *Directory
}

func NewFileCreatedEvent(connectionID connection_deck.ConnectionID, dir *Directory, file *File, opts ...event.Option) FileCreatedEvent {
	return FileCreatedEvent{
		event.NewBaseEvent(FileCreatedEventType, opts...),
		file,
		connectionID,
		dir,
	}
}

type FileCreatedSuccessEvent struct {
	event.BaseEvent
	File      *File
	Directory *Directory
}

func NewFileCreatedSuccessEvent(dir *Directory, file *File, opts ...event.Option) FileCreatedSuccessEvent {
	return FileCreatedSuccessEvent{
		event.NewBaseEvent(FileCreatedEventType.AsSuccess(), opts...),
		file,
		dir,
	}
}

type FileCreatedFailureEvent struct {
	event.BaseFailureEvent
	Directory *Directory
}

func NewFileCreatedFailureEvent(err error, dir *Directory) FileCreatedFailureEvent {
	return FileCreatedFailureEvent{
		event.NewBaseFailureEvent(FileCreatedEventType.AsFailure(), err),
		dir,
	}
}

type FileDeletedEvent struct {
	event.BaseEvent
	File            *File
	ConnectionID    connection_deck.ConnectionID
	ParentDirectory *Directory
}

type FileDeletedSuccessEvent struct {
	event.BaseEvent
	File            *File
	ParentDirectory *Directory
}

type FileDeletedFailureEvent struct {
	event.BaseFailureEvent
	ParentDirectory *Directory
}

func NewFileDeletedEvent(connectionID connection_deck.ConnectionID, parent *Directory, file *File, opts ...event.Option) FileDeletedEvent {
	return FileDeletedEvent{
		event.NewBaseEvent(FileDeletedEventType, opts...),
		file,
		connectionID,
		parent,
	}
}

func (e FileDeletedEvent) NewSuccessEvent(file *File, opts ...event.Option) FileDeletedSuccessEvent {
	return FileDeletedSuccessEvent{
		e.NewBaseSuccess(opts...),
		file,
		e.ParentDirectory,
	}
}

func (e FileDeletedEvent) NewFailureEvent(err error) FileDeletedFailureEvent {
	return FileDeletedFailureEvent{
		e.NewBaseFailure(err),
		e.ParentDirectory,
	}
}

func NewFileDeletedSuccessEvent(parent *Directory, file *File, opts ...event.Option) FileDeletedSuccessEvent {
	return FileDeletedSuccessEvent{
		event.NewBaseEvent(FileDeletedEventType.AsSuccess(), opts...),
		file,
		parent,
	}
}

func NewFileDeletedFailureEvent(err error, parent *Directory) FileDeletedFailureEvent {
	return FileDeletedFailureEvent{
		event.NewBaseFailureEvent(FileDeletedEventType.AsFailure(), err),
		parent,
	}
}

type FileLoadEvent struct {
	event.BaseEvent
	File         *File
	ConnectionID connection_deck.ConnectionID
}

func NewFileLoadEvent(connectionID connection_deck.ConnectionID, file *File, opts ...event.Option) FileLoadEvent {
	return FileLoadEvent{
		event.NewBaseEvent(FileLoadEventType, opts...),
		file,
		connectionID,
	}
}

type FileLoadSuccessEvent struct {
	event.BaseEvent
	File    *File
	Content FileContent
}

func NewFileLoadSuccessEvent(file *File, content FileContent) FileLoadSuccessEvent {
	return FileLoadSuccessEvent{
		event.NewBaseEvent(FileLoadEventType.AsSuccess()),
		file,
		content,
	}
}

type FileLoadFailureEvent struct {
	event.BaseFailureEvent
	File *File
}

func NewFileLoadFailureEvent(err error, file *File) FileLoadFailureEvent {
	return FileLoadFailureEvent{
		event.NewBaseFailureEvent(FileLoadEventType.AsFailure(), err),
		file,
	}
}

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
