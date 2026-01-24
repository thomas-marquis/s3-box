package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	ContentUploadedEventType event.Type = "event.content.uploaded"
	ContentDownloadEventType event.Type = "event.content.downloaded"
)

type withContent struct {
	content *Content
}

func (e withContent) Content() *Content {
	return e.content
}

type ContentUploadedEvent struct {
	event.BaseEvent
	withContent
	withDirectory
}

func NewContentUploadedEvent(directory *Directory, content *Content, opts ...event.Option) ContentUploadedEvent {
	return ContentUploadedEvent{
		event.NewBaseEvent(ContentUploadedEventType, opts...),
		withContent{content},
		withDirectory{directory},
	}
}

type ContentUploadedSuccessEvent struct {
	event.BaseEvent
	withFile
	withDirectory
}

func NewContentUploadedSuccessEvent(directory *Directory, file *File, opts ...event.Option) ContentUploadedSuccessEvent {
	return ContentUploadedSuccessEvent{
		event.NewBaseEvent(ContentUploadedEventType.AsSuccess(), opts...),
		withFile{file},
		withDirectory{directory},
	}
}

type ContentUploadedFailureEvent struct {
	event.BaseErrorEvent
	withDirectory
}

func NewContentUploadedFailureEvent(err error, dir *Directory) ContentUploadedFailureEvent {
	return ContentUploadedFailureEvent{
		event.NewBaseErrorEvent(ContentUploadedEventType.AsFailure(), err),
		withDirectory{dir},
	}
}

type ContentDownloadedEvent struct {
	event.BaseEvent
	withContent
	withConnectionID
}

func NewContentDownloadedEvent(connectionID connection_deck.ConnectionID, content *Content, opts ...event.Option) ContentDownloadedEvent {
	return ContentDownloadedEvent{
		event.NewBaseEvent(ContentDownloadEventType, opts...),
		withContent{content},
		withConnectionID{connectionID},
	}
}

type ContentDownloadedSuccessEvent struct {
	event.BaseEvent
	withContent
}

func NewContentDownloadedSuccessEvent(content *Content, opts ...event.Option) ContentDownloadedSuccessEvent {
	return ContentDownloadedSuccessEvent{
		event.NewBaseEvent(ContentDownloadEventType.AsSuccess(), opts...),
		withContent{content},
	}
}

type ContentDownloadedFailureEvent struct {
	event.BaseErrorEvent
}

func NewContentDownloadedFailureEvent(err error) ContentDownloadedFailureEvent {
	return ContentDownloadedFailureEvent{
		event.NewBaseErrorEvent(ContentDownloadEventType.AsFailure(), err),
	}
}
