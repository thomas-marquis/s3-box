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

type CreateSucceeded CreateTriggered

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

const (
	UploadTriggeredType event.Type = "event.directory.upload.triggered"
	UploadPreviewedType event.Type = "event.directory.upload.previewed"
	UploadConfirmedType event.Type = "event.directory.upload.confirmed"
	UploadAbortedType   event.Type = "event.directory.upload.aborted"
	UploadFailedType    event.Type = "event.directory.upload.failed"
	UploadSucceededType event.Type = "event.directory.upload.succeeded"
)

type UploadTriggered struct {
	Directory *Directory
	Items     []FsItem
}

func (e UploadTriggered) Type() event.Type {
	return UploadTriggeredType
}

type UploadPreviewed struct {
	Directory       *Directory
	Previews        map[UploadMode][]UploadedItemPreview
	UploadableItems []FsItem
}

func (e UploadPreviewed) Type() event.Type {
	return UploadPreviewedType
}

type UploadConfirmed struct {
	Directory       *Directory
	SelectedMode    UploadMode
	UploadableItems []FsItem
}

func (e UploadConfirmed) Type() event.Type {
	return UploadConfirmedType
}

type UploadAborted struct {
	Directory *Directory
}

func (e UploadAborted) Type() event.Type {
	return UploadAbortedType
}

type UploadFailed struct {
	Err       error
	Directory *Directory
}

func (e UploadFailed) Type() event.Type {
	return UploadFailedType
}

type UploadSucceeded LoadSucceeded

func (e UploadSucceeded) Type() event.Type {
	return UploadSucceededType
}

// on retourne une erreur direct avant le uploadTriggered si un sous-dossier existe déjà dans l'entité
// on refera un double check côté infra plus tard dés fois que l'état remote ait changé entre temps
// On ne retourne une preview et on ne demande une confirmation que si il y a plus de 1 fichier et/ou plus de 1 dossier
// On ne demande l'avis de l'utilisateur sur le mode d'upload que s'il y a conflit. On utilise celui par défaut sinon.
// Mais on lui montre la preview dans tous les cas et on lui demande confirmation
// On ne charge que le premier niveau si l'upload fonctionne
