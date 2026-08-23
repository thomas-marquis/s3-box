package state

import (
	"sync"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type TagsState struct {
	mu sync.RWMutex

	isLoading     binding.Bool
	statusLabel   binding.String
	currentTagSet *directory.TagSet
}

func newTagsState() *TagsState {
	return &TagsState{
		isLoading:   binding.NewBool(),
		statusLabel: binding.NewString(),
	}
}

func (t *TagsState) IsLoading() binding.Bool {
	return t.isLoading
}

func (t *TagsState) StatusLabel() binding.String {
	return t.statusLabel
}

func (t *TagsState) CurrentTagSet() *directory.TagSet {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentTagSet
}

func (t *TagsState) SetCurrentTagSet(ts *directory.TagSet) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentTagSet = ts
}

func (t *TagsState) Length() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.currentTagSet == nil {
		return 0
	}
	return len(t.currentTagSet.Get())
}
