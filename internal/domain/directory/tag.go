package directory

import (
	"errors"
	"maps"
	"slices"

	"github.com/thomas-marquis/it-happened/event"
)

var (
	ErrMaxTagCountReached  = errors.New("maximum tag count reached for this file")
	ErrTagKeyAlreadyExists = errors.New("tag key already exists")
	ErrTagKeyTooLong       = errors.New("tag key too long")
	ErrTagValueTooLong     = errors.New("tag value too long")
	ErrTagKeyNotExists     = errors.New("tag key not exists")
	ErrTagOperationPending = errors.New("a tag operation is already in pending")
)

const (
	MaxTagCount = 10
)

type Tag struct {
	Key   string
	Value string
}

type TagSet struct {
	tags        map[string]Tag
	pendingCmds []Command[Tag]
	pendingTags map[string]Tag
}

func (t *TagSet) Add(key, value string) error {
	if len(t.tags) >= MaxTagCount {
		return ErrMaxTagCountReached
	}
	if _, exists := t.tags[key]; exists {
		return ErrTagKeyAlreadyExists
	}
	for _, cmd := range t.pendingCmds {
		if cmd.Value().Key == key {
			return ErrTagOperationPending
		}
	}

	t.pendingCmds = append(t.pendingCmds, tagAdd{TagSet: t, NewTag: Tag{Key: key, Value: value}})
	return nil
}

func (t *TagSet) Remove(key string) error {
	return nil
}

func (t *TagSet) Update(key, value string) error {
	return nil
}

func (t *TagSet) Get() (tags []Tag) {
	return tagListFromMap(t.tags)
}

func (t *TagSet) Save() event.Event {
	if len(t.pendingCmds) == 0 {
		return event.New(event.ItHappened{}) // TODO: nothing happened
	}
	t.pendingTags = make(map[string]Tag)
	for _, cmd := range t.pendingCmds {
		cmd.Execute()
	}

	return event.New(TagsSaveTriggered{
		TagSet: t,
		Tags:   tagListFromMap(t.pendingTags),
	})
}

func (t *TagSet) Load() event.Event {
	return event.New(TagsLoadTriggered{TagSet: t})
}

func (t *TagSet) Notify(evt event.Event) {
	switch pl := evt.Payload().(type) {
	case TagsLoadSucceeded:
		t.replaceTags(pl.Tags)
	case TagsSaveSucceeded:
		t.replaceTags(pl.Tags)
	}
}

func (t *TagSet) replaceTags(newTags []Tag) {
	t.tags = make(map[string]Tag)
	for _, tag := range newTags {
		t.tags[tag.Key] = tag
	}
}

type Command[T any] interface {
	Value() T
	Execute()
}

type tagAdd struct {
	NewTag Tag
	TagSet *TagSet
}

func (c tagAdd) Value() Tag {
	return c.NewTag
}

func (c tagAdd) Execute() {
	c.TagSet.pendingTags[c.NewTag.Key] = c.NewTag
}

func tagListFromMap(tags map[string]Tag) []Tag {
	keys := slices.Collect(maps.Keys(tags))
	slices.Sort(keys)
	var tagList []Tag
	for _, key := range keys {
		tagList = append(tagList, tags[key])
	}
	return tagList
}
