package directory

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/u"
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
	MaxTagCount       = 10
	MaxTagKeyLength   = 128
	MaxTagValueLength = 256
)

type Tag struct {
	Key   string
	Value string
}

func CompareTag(t1, t2 Tag) bool {
	return t1.Key == t2.Key && t1.Value == t2.Value
}

func NewTag(key, value string) (Tag, error) {
	if err := validateTag(key, value); err != nil {
		return Tag{}, err
	}
	return Tag{Key: key, Value: value}, nil
}

func validateTag(key, value string) error {
	if len(key) > MaxTagKeyLength {
		return ErrTagKeyTooLong
	}
	if len(value) > MaxTagValueLength {
		return ErrTagValueTooLong
	}
	return nil
}

type TagSet struct {
	*u.Observable[[]Tag]

	mu sync.RWMutex

	tags         map[string]Tag
	pendingCmds  []Command
	objectPath   Path
	connectionID connection_deck.ConnectionID
	isLoaded     *u.ObservableValue[bool]
}

func NewTagSet(path Path, connectionID connection_deck.ConnectionID) *TagSet {
	return &TagSet{
		Observable:   u.NewObservable[[]Tag](),
		tags:         make(map[string]Tag),
		pendingCmds:  make([]Command, 0),
		objectPath:   path,
		connectionID: connectionID,
		isLoaded:     u.NewObservableValue(false),
	}
}

func (t *TagSet) ID() string {
	return fmt.Sprintf("%s:%s", t.connectionID, t.objectPath)
}

func (t *TagSet) ObjectPath() Path {
	return t.objectPath
}

func (t *TagSet) IsLoaded() *u.ObservableValue[bool] {
	return t.isLoaded
}

func (t *TagSet) ConnectionID() connection_deck.ConnectionID {
	return t.connectionID
}

func (t *TagSet) HasPending() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.pendingCmds) > 0
}

func (t *TagSet) Add(key, value string) error {
	t.mu.RLock()
	if len(t.tags) >= MaxTagCount {
		return ErrMaxTagCountReached
	}
	if err := validateTag(key, value); err != nil {
		return err
	}
	if _, exists := t.tags[key]; exists {
		return ErrTagKeyAlreadyExists
	}

	for _, cmd := range t.pendingCmds {
		if cmd.Key() == key {
			return ErrTagOperationPending
		}
	}
	t.mu.RUnlock()

	t.mu.Lock()
	t.pendingCmds = append(t.pendingCmds, tagAdd{TagSet: t, NewTag: Tag{Key: key, Value: value}})
	t.mu.Unlock()
	return nil
}

func (t *TagSet) Remove(key string) error {
	if len(key) > MaxTagKeyLength {
		return ErrTagKeyTooLong
	}
	t.mu.RLock()
	if _, exists := t.tags[key]; !exists {
		return ErrTagKeyNotExists
	}
	for _, cmd := range t.pendingCmds {
		if cmd.Key() == key {
			return ErrTagOperationPending
		}
	}
	t.mu.RUnlock()

	t.mu.Lock()
	t.pendingCmds = append(t.pendingCmds, tagRemove{TagSet: t, TagKey: key})
	t.mu.Unlock()
	return nil
}

func (t *TagSet) Update(originalKey, newKey, newValue string) error {
	if err := validateTag(newKey, newValue); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.tags[originalKey]; !exists {
		return ErrTagKeyNotExists
	}

	if _, exists := t.tags[newKey]; newKey != originalKey && exists {
		return ErrTagKeyAlreadyExists
	}

	for i, cmd := range t.pendingCmds {
		if cmd.Key() == originalKey {
			if _, ok := cmd.(tagUpdate); ok {
				t.pendingCmds[i] = tagUpdate{TagKey: originalKey, NewKey: newKey, NewValue: newValue}
				return nil
			}
			return ErrTagOperationPending
		}
	}

	t.pendingCmds = append(t.pendingCmds, tagUpdate{TagKey: originalKey, NewKey: newKey, NewValue: newValue})
	return nil
}

func (t *TagSet) Get() (tags []Tag) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return tagListFromMap(t.tags)
}

func (t *TagSet) Save() event.Event {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.pendingCmds) == 0 {
		return event.New(event.NothingHappened{})
	}
	pendingTags := make(map[string]Tag)
	for k, v := range t.tags {
		pendingTags[k] = v
	}
	for _, cmd := range t.pendingCmds {
		cmd.Execute(pendingTags)
	}

	clear(t.pendingCmds)

	return event.New(TagsSaveTriggered{
		TagSet: t,
		Tags:   tagListFromMap(pendingTags),
	})
}

func (t *TagSet) Load() event.Event {
	if t.isLoaded.Get() {
		return event.New(event.ItHappened{}) // TODO: nothing happened
	}
	return event.New(TagsLoadTriggered{TagSet: t})
}

func (t *TagSet) Notify(evt event.Event) {
	switch pl := evt.Payload().(type) {
	case TagsLoadSucceeded:
		t.replaceTags(pl.Tags)
		t.TriggerAll(pl.Tags)
		t.isLoaded.Set(true)
	case TagsSaveSucceeded:
		t.replaceTags(pl.Tags)
		t.TriggerAll(pl.Tags)
	}
}

func (t *TagSet) replaceTags(newTags []Tag) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tags = make(map[string]Tag)
	for _, tag := range newTags {
		t.tags[tag.Key] = tag
	}
}

type Command interface {
	Key() string
	Execute(acc map[string]Tag)
}

type tagAdd struct {
	NewTag Tag
	TagSet *TagSet
}

func (c tagAdd) Key() string {
	return c.NewTag.Key
}

func (c tagAdd) Execute(acc map[string]Tag) {
	acc[c.NewTag.Key] = c.NewTag
}

type tagRemove struct {
	TagKey string
	TagSet *TagSet
}

func (c tagRemove) Key() string {
	return c.TagKey
}

func (c tagRemove) Execute(acc map[string]Tag) {
	delete(acc, c.TagKey)
}

type tagUpdate struct {
	TagKey   string
	NewKey   string
	NewValue string
}

func (c tagUpdate) Key() string {
	return c.TagKey
}

func (c tagUpdate) Execute(acc map[string]Tag) {
	acc[c.TagKey] = Tag{Key: c.NewKey, Value: c.NewValue}
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
