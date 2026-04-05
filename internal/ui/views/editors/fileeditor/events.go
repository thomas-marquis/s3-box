package fileeditor

import (
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	SaveTriggeredType event.Type = "event.fileeditor.save.triggered"
	SaveSucceededType event.Type = "event.fileeditor.save.succeeded"
	SaveFailedType    event.Type = "event.fileeditor.save.failed"
)

type SaveTriggered struct {
	File    *directory.File
	Content string
}

func (e SaveTriggered) Type() event.Type {
	return SaveTriggeredType
}

type SaveSucceeded struct {
	File    *directory.File
	Content string
}

func (e SaveSucceeded) Type() event.Type {
	return SaveSucceededType
}

type SaveFailed struct {
	Err  error
	File *directory.File
}

func (e SaveFailed) Type() event.Type {
	return SaveFailedType
}
