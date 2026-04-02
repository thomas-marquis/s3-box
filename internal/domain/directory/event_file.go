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

type withSrcPath struct {
	srcPath string
}

func (e withSrcPath) SrcPath() string {
	return e.srcPath
}

type withDstPath struct {
	dstPath string
}

func (e withDstPath) DstPath() string {
	return e.dstPath
}

type FileCreatedEvent struct {
	event.BaseEvent
	withFile
	withConnectionID
	withDirectory
}

func NewFileCreatedEvent(connectionID connection_deck.ConnectionID, dir *Directory, file *File, opts ...event.Option) FileCreatedEvent {
	return FileCreatedEvent{
		event.NewBaseEvent(FileCreatedEventType, opts...),
		withFile{file},
		withConnectionID{connectionID},
		withDirectory{dir},
	}
}

type FileCreatedSuccessEvent struct {
	event.BaseEvent
	withFile
	withDirectory
}

func NewFileCreatedSuccessEvent(dir *Directory, file *File, opts ...event.Option) FileCreatedSuccessEvent {
	return FileCreatedSuccessEvent{
		event.NewBaseEvent(FileCreatedEventType.AsSuccess(), opts...),
		withFile{file},
		withDirectory{dir},
	}
}

type FileCreatedFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
}

func NewFileCreatedFailureEvent(err error, dir *Directory) FileCreatedFailureEvent {
	return FileCreatedFailureEvent{
		event.NewBaseErrorEvent(FileCreatedEventType.AsFailure(), err),
		withDirectory{dir},
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
	Content FileContent
}

func NewFileLoadSuccessEvent(file *File, content FileContent) FileLoadSuccessEvent {
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

type FileRenameEvent struct {
	event.BaseEvent
	withFile
	withNewName
	withDirectory
}

func NewFileRenameEvent(dir *Directory, file *File, newName string, opts ...event.Option) FileRenameEvent {
	return FileRenameEvent{
		event.NewBaseEvent(FileRenameEventType, opts...),
		withFile{file},
		withNewName{newName},
		withDirectory{dir},
	}
}

type FileRenameSuccessEvent struct {
	event.BaseEvent
	withFile
	withNewName
	withDirectory
}

func NewFileRenameSuccessEvent(dir *Directory, file *File, newName string, opts ...event.Option) FileRenameSuccessEvent {
	return FileRenameSuccessEvent{
		event.NewBaseEvent(FileRenameEventType.AsSuccess(), opts...),
		withFile{file},
		withNewName{newName},
		withDirectory{dir},
	}
}

type FileRenameFailureEvent struct {
	event.BaseErrorEvent
	withFile
	withNewName
	withDirectory
}

func NewFileRenameFailureEvent(err error, dir *Directory, file *File, newName string) FileRenameFailureEvent {
	return FileRenameFailureEvent{
		event.NewBaseErrorEvent(FileRenameEventType.AsFailure(), err),
		withFile{file},
		withNewName{newName},
		withDirectory{dir},
	}
}

type FileUploadEvent struct {
	event.BaseEvent
	withDirectory
	withSrcPath
}

func NewFileUploadEvent(directory *Directory, localFilePath string, opts ...event.Option) FileUploadEvent {
	return FileUploadEvent{
		event.NewBaseEvent(FileUploadEventType, opts...),
		withDirectory{directory},
		withSrcPath{localFilePath},
	}
}

type FileUploadSuccessEvent struct {
	event.BaseEvent
	withFile
	withDirectory
}

func NewFileUploadSuccessEvent(directory *Directory, file *File, opts ...event.Option) FileUploadSuccessEvent {
	return FileUploadSuccessEvent{
		event.NewBaseEvent(FileUploadEventType.AsSuccess(), opts...),
		withFile{file},
		withDirectory{directory},
	}
}

type FileUploadFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
}

func NewFileUploadFailureEvent(err error, dir *Directory) FileUploadFailureEvent {
	return FileUploadFailureEvent{
		event.NewBaseErrorEvent(FileUploadEventType.AsFailure(), err),
		withDirectory{dir},
	}
}

type FileDownloadEvent struct {
	event.BaseEvent
	withConnectionID
	withDstPath
	withFile
}

func NewFileDownloadEvent(connectionID connection_deck.ConnectionID, file *File, dstLocalPtah string, opts ...event.Option) FileDownloadEvent {
	return FileDownloadEvent{
		event.NewBaseEvent(FileDownloadEventType, opts...),
		withConnectionID{connectionID},
		withDstPath{dstLocalPtah},
		withFile{file},
	}
}

type FileDownloadSuccessEvent struct {
	event.BaseEvent
	withFile
}

func NewFileDownloadSuccessEvent(file *File, opts ...event.Option) FileDownloadSuccessEvent {
	return FileDownloadSuccessEvent{
		event.NewBaseEvent(FileDownloadEventType.AsSuccess(), opts...),
		withFile{file},
	}
}

type FileDownloadFailureEvent struct {
	event.BaseErrorEvent
}

func NewFileDownloadFailureEvent(err error) FileDownloadFailureEvent {
	return FileDownloadFailureEvent{
		event.NewBaseErrorEvent(FileDownloadEventType.AsFailure(), err),
	}
}
