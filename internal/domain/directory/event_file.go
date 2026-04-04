package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreateFileTriggeredType event.Type = "event.file.create.triggered"
	CreateFileSucceededType event.Type = "event.file.create.succeeded"
	CreateFileFailedType    event.Type = "event.file.create.failed"
)

type CreateFileTriggered struct {
	File         *File
	ConnectionID connection_deck.ConnectionID
	Directory    *Directory
}

func (e CreateFileTriggered) Type() event.Type {
	return CreateFileTriggeredType
}

type CreateFileSucceeded struct {
	File      *File
	Directory *Directory
}

func (e CreateFileSucceeded) Type() event.Type {
	return CreateFileSucceededType
}

type CreateFileFailed struct {
	Err       error
	Directory *Directory
}

func (e CreateFileFailed) Type() event.Type {
	return CreateFileFailedType
}

const (
	DeleteFileTriggeredType event.Type = "event.file.delete.triggered"
	DeleteFileSucceededType event.Type = "event.file.delete.succeeded"
	DeleteFileFailedType    event.Type = "event.file.delete.failed"
)

type DeleteFileTriggered struct {
	File            *File
	ConnectionID    connection_deck.ConnectionID
	ParentDirectory *Directory
}

func (e DeleteFileTriggered) Type() event.Type {
	return DeleteFileTriggeredType
}

type DeleteFileSucceeded struct {
	File            *File
	ParentDirectory *Directory
}

func (e DeleteFileSucceeded) Type() event.Type {
	return DeleteFileSucceededType
}

type DeleteFileFailed struct {
	Err             error
	ParentDirectory *Directory
}

func (e DeleteFileFailed) Type() event.Type {
	return DeleteFileFailedType
}

const (
	LoadFileTriggeredType event.Type = "event.file.load.triggered"
	LoadFileSucceededType event.Type = "event.file.load.succeeded"
	LoadFileFailedType    event.Type = "event.file.load.failed"
)

type LoadFileTriggered struct {
	File         *File
	ConnectionID connection_deck.ConnectionID
}

func (e LoadFileTriggered) Type() event.Type {
	return LoadFileTriggeredType
}

func NewFileLoadEvent(connectionID connection_deck.ConnectionID, file *File, opts ...event.Option) LoadFileTriggered {
	return LoadFileTriggered{
		file,
		connectionID,
	}
}

type LoadFileSucceeded struct {
	File    *File
	Content FileContent
}

func (e LoadFileSucceeded) Type() event.Type {
	return LoadFileSucceededType
}

func NewFileLoadSuccessEvent(file *File, content FileContent) LoadFileSucceeded {
	return LoadFileSucceeded{
		file,
		content,
	}
}

type LoadFileFailed struct {
	Err  error
	File *File
}

func (e LoadFileFailed) Type() event.Type {
	return LoadFileFailedType
}

const (
	RenameFileTriggeredType event.Type = "event.file.rename.triggered"
	RenameFileSucceededType event.Type = "event.file.rename.succeeded"
	RenameFileFailedType    event.Type = "event.file.rename.failed"
)

type RenameFileTriggered struct {
	File      *File
	NewName   string
	Directory *Directory
}

func (e RenameFileTriggered) Type() event.Type {
	return RenameFileTriggeredType
}

type RenameFileSucceeded struct {
	File      *File
	NewName   string
	Directory *Directory
}

func (e RenameFileSucceeded) Type() event.Type {
	return RenameFileSucceededType
}

type RenameFileFailed struct {
	Err       error
	File      *File
	NewName   string
	Directory *Directory
}

func (e RenameFileFailed) Type() event.Type {
	return RenameFileFailedType
}

const (
	UploadFileTriggeredType event.Type = "event.file.upload.triggered"
	UploadFileSucceededType event.Type = "event.file.upload.succeeded"
	UploadFileFailedType    event.Type = "event.file.upload.failed"
)

type UploadFileTriggered struct {
	Directory *Directory
	SrcPath   string
}

func (e UploadFileTriggered) Type() event.Type {
	return UploadFileTriggeredType
}

type UploadFileSucceeded struct {
	File      *File
	Directory *Directory
}

func (e UploadFileSucceeded) Type() event.Type {
	return UploadFileSucceededType
}

type UploadFileFailed struct {
	Err       error
	Directory *Directory
}

func (e UploadFileFailed) Type() event.Type {
	return UploadFileFailedType
}

const (
	DownloadFileTriggeredType event.Type = "event.file.download.triggered"
	DownloadFileSucceededType event.Type = "event.file.download.succeeded"
	DownloadFileFailedType    event.Type = "event.file.download.failed"
)

type DownloadFileTriggered struct {
	ConnectionID connection_deck.ConnectionID
	DstPath      string
	File         *File
}

func (e DownloadFileTriggered) Type() event.Type {
	return DownloadFileTriggeredType
}

type DownloadFileSucceeded struct {
	File *File
}

func (e DownloadFileSucceeded) Type() event.Type {
	return DownloadFileSucceededType
}

type DownloadFileFailed struct {
	Err error
}

func (e DownloadFileFailed) Type() event.Type {
	return DownloadFileFailedType
}
