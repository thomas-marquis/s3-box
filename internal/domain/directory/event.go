package directory

import (
	"context"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

const (
	CreatedEventName         = "Event.directory.created"
	DeletedEventName         = "Event.directory.deleted"
	FileCreatedEventName     = "Event.file.created"
	FileDeletedEventName     = "Event.file.deleted"
	ContentUploadedEventName = "Event.content.uploaded"
	ContentDownloadEventName = "Event.content.downloaded"
)

type Event interface {
	Name() string
	ConnectionID() connection_deck.ConnectionID
	AttachSuccessCallback(func())
	AttachErrorCallback(func(error))
	CallSuccessCallbacks()
	CallErrorCallbacks(error)
	AttachContext(context.Context)
	Context() context.Context
}

type baseEvent struct {
	connectionID     connection_deck.ConnectionID
	successCallbacks []func()
	errorCallbacks   []func(error)
	context          context.Context
}

func (e baseEvent) ConnectionID() connection_deck.ConnectionID {
	return e.connectionID
}

func (e baseEvent) AttachSuccessCallback(callback func()) {
	e.successCallbacks = append(e.successCallbacks, callback)
}

func (e baseEvent) AttachErrorCallback(callback func(error)) {
	e.errorCallbacks = append(e.errorCallbacks, callback)
}

func (e baseEvent) CallSuccessCallbacks() {
	if e.successCallbacks == nil {
		return
	}
	for _, callback := range e.successCallbacks {
		callback()
	}
}

func (e baseEvent) CallErrorCallbacks(err error) {
	if err == nil || e.errorCallbacks == nil {
		return
	}
	for _, callback := range e.errorCallbacks {
		callback(err)
	}
}

func (e baseEvent) AttachContext(ctx context.Context) {
	e.context = ctx
}

func (e baseEvent) Context() context.Context {
	if e.context == nil {
		return context.Background()
	}
	return e.context
}
