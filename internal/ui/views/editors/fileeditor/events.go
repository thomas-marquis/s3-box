package fileeditor

import (
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	SaveEventType event.Type = "event.fileeditor.save"
)

type SaveEvent struct {
	event.BaseEvent
	File    *directory.File
	Content string
}

func NewSaveEvent(file *directory.File, content string, options ...event.Option) SaveEvent {
	return SaveEvent{
		BaseEvent: event.NewBaseEvent(SaveEventType, options...),
		File:      file,
		Content:   content,
	}
}

type SaveSuccessEvent struct {
	event.BaseEvent
	File    *directory.File
	Content string
}

func NewSaveSuccessEvent(file *directory.File, content string, options ...event.Option) SaveSuccessEvent {
	return SaveSuccessEvent{
		BaseEvent: event.NewBaseEvent(SaveEventType.AsSuccess(), options...),
		File:      file,
		Content:   content,
	}
}

type SaveFailureEvent struct {
	event.BaseFailureEvent
	File *directory.File
}

func NewSaveFailureEvent(file *directory.File, err error) SaveFailureEvent {
	return SaveFailureEvent{
		BaseFailureEvent: event.NewBaseFailureEvent(SaveEventType.AsFailure(), err),
		File:             file,
	}
}
