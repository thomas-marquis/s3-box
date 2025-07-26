package directory

import "github.com/thomas-marquis/s3-box/internal/domain/connection_deck"

type ContentEvent interface {
	Event
	Content() *Content
}

type contentUploadedEvent struct {
	baseEvent
	content *Content
}

var _ ContentEvent = &contentUploadedEvent{}

func newContentUploadedEvent(connectionID connection_deck.ConnectionID, content *Content) contentUploadedEvent {
	return contentUploadedEvent{baseEvent{connectionID, nil, nil, nil}, content}
}

func (e contentUploadedEvent) Name() string {
	return ContentUploadedEventName
}

func (e contentUploadedEvent) Content() *Content {
	return e.content
}

type contentDownloadedEvent struct {
	baseEvent
	content *Content
}

var _ ContentEvent = &contentDownloadedEvent{}

func newContentDownloadedEvent(connectionID connection_deck.ConnectionID, content *Content) contentDownloadedEvent {
	return contentDownloadedEvent{baseEvent{connectionID, nil, nil, nil}, content}
}

func (e contentDownloadedEvent) Name() string {
	return ContentDownloadEventName
}

func (e contentDownloadedEvent) Content() *Content {
	return e.content
}
