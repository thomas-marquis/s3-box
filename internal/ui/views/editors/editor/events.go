package editor

import (
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

const (
	SaveTriggeredType event.Type = "event.fileeditor.save.triggered"
	SaveSucceededType event.Type = "event.fileeditor.save.succeeded"
	SaveFailedType    event.Type = "event.fileeditor.save.failed"

	/////

	LoadedType         event.Type = "event.editor.loaded"
	LoadFailedType     event.Type = "event.editor.load.failed"
	ClosedType         event.Type = "event.editor.closed"
	CloseRequestedType event.Type = "event.editor.close.requested"
	CloseConfirmedType event.Type = "event.editor.close.confirmed"
	CloseCanceledType  event.Type = "event.editor.close.canceled"
)

type Payload interface {
	This() Editor
}

type Loaded struct {
	Editor  Editor
	Content directory.FileContent
}

func (Loaded) EventType() event.Type {
	return LoadedType
}

func (p Loaded) This() Editor {
	return p.Editor
}

type LoadFailed struct {
	Editor Editor
	Err    error
}

func (LoadFailed) EventType() event.Type {
	return LoadFailedType
}

func (p LoadFailed) This() Editor {
	return p.Editor
}

type Closed struct {
	Editor Editor
}

func (Closed) EventType() event.Type {
	return ClosedType
}

func (p Closed) This() Editor {
	return p.Editor
}

type CloseRequested struct {
	Editor Editor
	cancel func()
}

func NewCloseRequested(e Editor, cancel func()) CloseRequested {
	return CloseRequested{Editor: e, cancel: cancel}
}

func (CloseRequested) EventType() event.Type {
	return CloseRequestedType
}

func (p CloseRequested) This() Editor {
	return p.Editor
}

func (p CloseRequested) Confirm(evt event.Event) event.Event {
	return evt.NewFollowup(CloseConfirmed{Editor: p.Editor})
}

func (p CloseRequested) Cancel(evt event.Event) event.Event {
	if p.cancel != nil {
		p.cancel()
	}
	return evt.NewFollowup(CloseCanceled{Editor: p.Editor})
}

type CloseConfirmed struct {
	Editor Editor
}

func (CloseConfirmed) EventType() event.Type {
	return CloseConfirmedType
}

func (p CloseConfirmed) This() Editor {
	return p.Editor
}

type CloseCanceled struct {
	Editor Editor
}

func (CloseCanceled) EventType() event.Type {
	return CloseCanceledType
}

func (p CloseCanceled) This() Editor {
	return p.Editor
}

/////

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
