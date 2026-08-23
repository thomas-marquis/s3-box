package directory

import (
	"github.com/thomas-marquis/it-happened/event"
)

const (
	TagsSaveTriggeredType event.Type = "event.tag.save.triggered"
	TagsSaveSucceededType event.Type = "event.tag.save.succeeded"
	TagsSaveFailedType    event.Type = "event.tag.save.failed"
)

type TagsSaveTriggered struct {
	TagSet *TagSet
	Tags   []Tag
}

func (p TagsSaveTriggered) EventType() event.Type {
	return TagsSaveTriggeredType
}

type TagsSaveSucceeded struct {
	TagSet *TagSet
	Tags   []Tag
}

func (p TagsSaveSucceeded) EventType() event.Type {
	return TagsSaveSucceededType
}

type TagsSaveFailed struct {
	TagSet *TagSet
	Error  error
}

func (p TagsSaveFailed) EventType() event.Type {
	return TagsSaveFailedType
}

const (
	TagsLoadTriggeredType event.Type = "event.tag.load.triggered"
	TagsLoadSucceededType event.Type = "event.tag.load.succeeded"
	TagsLoadFailedType    event.Type = "event.tag.load.failed"
)

type TagsLoadTriggered struct {
	TagSet *TagSet
}

func (p TagsLoadTriggered) EventType() event.Type {
	return TagsLoadTriggeredType
}

type TagsLoadSucceeded struct {
	TagSet *TagSet
	Tags   []Tag
}

func (p TagsLoadSucceeded) EventType() event.Type {
	return TagsLoadSucceededType
}

type TagsLoadFailed struct {
	TagSet *TagSet
	Error  error
}

func (p TagsLoadFailed) EventType() event.Type {
	return TagsLoadFailedType
}
