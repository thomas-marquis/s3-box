package directory

import (
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
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
	//Recursive    bool
}

func (e CreateFileTriggered) EventType() event.Type {
	return CreateFileTriggeredType
}

type CreateFileSucceeded struct {
	File      *File
	Directory *Directory
}

func (e CreateFileSucceeded) EventType() event.Type {
	return CreateFileSucceededType
}

type CreateFileFailed struct {
	Err       error
	Directory *Directory
}

func (e CreateFileFailed) EventType() event.Type {
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

func (e DeleteFileTriggered) EventType() event.Type {
	return DeleteFileTriggeredType
}

type DeleteFileSucceeded struct {
	File            *File
	ParentDirectory *Directory
}

func (e DeleteFileSucceeded) EventType() event.Type {
	return DeleteFileSucceededType
}

type DeleteFileFailed struct {
	Err             error
	ParentDirectory *Directory
}

func (e DeleteFileFailed) EventType() event.Type {
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

func (e LoadFileTriggered) EventType() event.Type {
	return LoadFileTriggeredType
}

type LoadFileSucceeded struct {
	File    *File
	Content FileContent
}

func (e LoadFileSucceeded) EventType() event.Type {
	return LoadFileSucceededType
}

type LoadFileFailed struct {
	Err  error
	File *File
}

func (e LoadFileFailed) EventType() event.Type {
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

func (e RenameFileTriggered) EventType() event.Type {
	return RenameFileTriggeredType
}

type RenameFileSucceeded struct {
	File      *File
	NewName   string
	Directory *Directory
}

func (e RenameFileSucceeded) EventType() event.Type {
	return RenameFileSucceededType
}

type RenameFileFailed struct {
	Err       error
	File      *File
	NewName   string
	Directory *Directory
}

func (e RenameFileFailed) EventType() event.Type {
	return RenameFileFailedType
}

const (
	UploadFileTriggeredType event.Type = "event.file.upload.triggered"
	UploadFileSucceededType event.Type = "event.file.upload.succeeded"
	UploadFileFailedType    event.Type = "event.file.upload.failed"

	UploadMultipleFilesSucceededType event.Type = "event.file.upload.multiple.succeeded"
	UploadMultipleFilesFailedType    event.Type = "event.file.upload.multiple.failed"
)

type UploadFileTriggered struct {
	Directory *Directory
	SrcPath   string
}

func (e UploadFileTriggered) EventType() event.Type {
	return UploadFileTriggeredType
}

type UploadFileSucceeded struct {
	File      *File
	Directory *Directory
}

func (e UploadFileSucceeded) EventType() event.Type {
	return UploadFileSucceededType
}

type UploadFileFailed struct {
	Err       error
	Directory *Directory
}

func (e UploadFileFailed) EventType() event.Type {
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

func (e DownloadFileTriggered) EventType() event.Type {
	return DownloadFileTriggeredType
}

type DownloadFileSucceeded struct {
	File *File
}

func (e DownloadFileSucceeded) EventType() event.Type {
	return DownloadFileSucceededType
}

type DownloadFileFailed struct {
	Err error
}

func (e DownloadFileFailed) EventType() event.Type {
	return DownloadFileFailedType
}

type UploadMultipleFilesSucceeded struct {
	Files     []*File
	Directory *Directory
}

func (e UploadMultipleFilesSucceeded) EventType() event.Type {
	return UploadMultipleFilesSucceededType
}

type UploadMultipleFilesFailed struct {
	Err error
}

func (e UploadMultipleFilesFailed) EventType() event.Type {
	return UploadMultipleFilesFailedType
}
