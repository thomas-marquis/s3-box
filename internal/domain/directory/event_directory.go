package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	CreateTriggeredType event.Type = "event.directory.create.treiggered"
	CreateSucceededType event.Type = "event.directory.create.succeeded"
	CreateFailedType    event.Type = "event.directory.create.failed"
)

type CreateTriggered struct {
	ParentDirectory *Directory
	Directory       *Directory
}

func (e CreateTriggered) Type() event.Type {
	return CreateTriggeredType
}

type CreateSucceeded struct {
	ParentDirectory *Directory
	Directory       *Directory
}

func (e CreateSucceeded) Type() event.Type {
	return CreateSucceededType
}

type CreateFailed struct {
	Err             error
	ParentDirectory *Directory
}

func (e CreateFailed) Type() event.Type {
	return CreateFailedType
}

const (
	DeleteTriggeredType event.Type = "event.directory.delete.triggered"
	DeleteSucceededType event.Type = "event.directory.delete.succeeded"
	DeleteFailedType    event.Type = "event.directory.delete.failed"
)

type DeleteTriggered struct {
	Directory      *Directory
	DeletedDirPath Path
}

func (e DeleteTriggered) Type() event.Type {
	return DeleteTriggeredType
}

type DeleteSucceeded struct {
	Directory *Directory
}

func (e DeleteSucceeded) Type() event.Type {
	return DeleteSucceededType
}

type DeleteFailed struct {
	Err error
}

func (e DeleteFailed) Type() event.Type {
	return DeleteFailedType
}

const (
	LoadTriggeredType event.Type = "event.directory.load.triggered"
	LoadSucceededType event.Type = "event.directory.load.succeeded"
	LoadFailedType    event.Type = "event.directory.load.failed"
)

type LoadTriggered struct {
	Directory *Directory
}

func (e LoadTriggered) Type() event.Type {
	return LoadTriggeredType
}

type LoadSucceeded struct {
	Directory      *Directory
	Files          []*File
	SubDirectories []*Directory
}

func (e LoadSucceeded) Type() event.Type {
	return LoadSucceededType
}

type LoadFailed struct {
	Err       error
	Directory *Directory
}

func (e LoadFailed) Type() event.Type {
	return LoadFailedType
}

const (
	RenameTriggeredType event.Type = "event.directory.rename.triggered"
	RenameSucceededType event.Type = "event.directory.rename.succeeded"
	RenameFailedType    event.Type = "event.directory.rename.failed"
)

type RenameTriggered struct {
	Directory *Directory
	NewName   string
}

func (e RenameTriggered) Type() event.Type {
	return RenameTriggeredType
}

type RenameSucceeded struct {
	Directory *Directory
	NewName   string
}

func (e RenameSucceeded) Type() event.Type {
	return RenameSucceededType
}

type RenameFailed struct {
	Err       error
	Directory *Directory
	NewName   string
}

func (e RenameFailed) Type() event.Type {
	return RenameFailedType
}

const (
	RenameRecoveryTriggeredType event.Type = "event.directory.rename.recovery.triggered"
)

type RenameRecoveryTriggered struct {
	Directory *Directory
	DstDir    *Directory
	Choice    RecoveryChoice
}

func (e RenameRecoveryTriggered) Type() event.Type {
	return RenameRecoveryTriggeredType
}

const (
	UserValidationAskedType    event.Type = "event.directory.user.validation.asked"
	UserValidationAcceptedType event.Type = "event.directory.user.validation.accepted"
	UserValidationRefusedType  event.Type = "event.directory.user.validation.refused"
)

type UserValidationAsked struct {
	Directory *Directory
	Reason    event.Event
	Message   string
}

func (e UserValidationAsked) Type() event.Type {
	return UserValidationAskedType
}

type UserValidationAccepted struct {
	Directory *Directory
	Reason    event.Event
}

func (e UserValidationAccepted) Type() event.Type {
	return UserValidationAcceptedType
}

type UserValidationRefused struct {
	Directory *Directory
	Reason    event.Event
}

func (e UserValidationRefused) Type() event.Type {
	return UserValidationRefusedType
}
