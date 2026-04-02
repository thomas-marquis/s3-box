package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	FileUploadEventType   event.Type = "event.file.upload"
	FileDownloadEventType event.Type = "event.file.download"
)

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
