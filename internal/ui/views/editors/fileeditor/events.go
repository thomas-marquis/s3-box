package fileeditor

import (
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
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

func (e SaveTriggered) EventType() event.Type {
	return SaveTriggeredType
}

type SaveSucceeded struct {
	File    *directory.File
	Content string
}

func (e SaveSucceeded) EventType() event.Type {
	return SaveSucceededType
}

type SaveFailed struct {
	Err  error
	File *directory.File
}

func (e SaveFailed) EventType() event.Type {
	return SaveFailedType
}
