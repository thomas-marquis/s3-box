package viewmodel

import (
	"context"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
)

type TagsViewmodel interface {
	Select(ts *directory.TagSet)
	Save()
	Add(key, value string) error
}

type tagsViewmodelImpl struct {
	bus   event.Bus
	state *state.TagsState
}

func NewTagsViewmodel(ctx context.Context, bus event.Bus, appState *state.State) TagsViewmodel {
	v := &tagsViewmodelImpl{
		state: appState.Tags(),
		bus:   bus,
	}

	sub := bus.Subscribe().
		On(event.Is(directory.TagsLoadSucceededType), v.handleLoaded).
		On(event.Is(directory.TagsLoadFailedType), v.handleLoadFailed).
		On(event.Is(directory.TagsSaveSucceededType), v.handleSaved).
		On(event.Is(directory.TagsSaveFailedType), v.handleSaveFailed)
	sub.ListenNonBlocking()

	go func() {
		<-ctx.Done()
		bus.Unsubscribe(sub)
	}()

	return v
}

func (v *tagsViewmodelImpl) Select(ts *directory.TagSet) {
	u.Skip(v.state.IsLoading().Set(true))
	u.Skip(v.state.StatusLabel().Set(""))
	v.state.SetCurrentTagSet(ts)
	v.bus.Publish(ts.Load())
}

func (v *tagsViewmodelImpl) Save() {
	u.Skip(v.state.StatusLabel().Set("Saving..."))
	v.bus.Publish(v.state.CurrentTagSet().Save())
}

func (v *tagsViewmodelImpl) Add(key, value string) error {
	ts := v.state.CurrentTagSet()
	if err := ts.Add(key, value); err != nil {
		return err
	}
	v.bus.Publish(ts.Save())
	return nil
}

func (v *tagsViewmodelImpl) handleLoaded(evt event.Event) {
	pl := evt.Payload().(directory.TagsLoadSucceeded)
	pl.TagSet.Notify(evt)
	if v.hasCurrentTagSetChanged(pl.TagSet) {
		return
	}
	u.Skip(v.state.IsLoading().Set(false))
}

func (v *tagsViewmodelImpl) handleLoadFailed(evt event.Event) {
	pl := evt.Payload().(directory.TagsLoadFailed)
	pl.TagSet.Notify(evt)
	if v.hasCurrentTagSetChanged(pl.TagSet) {
		return
	}
	u.Skip(v.state.IsLoading().Set(false))
	u.Skip(v.state.StatusLabel().Set("Tag loading failed"))
}

func (v *tagsViewmodelImpl) handleSaved(evt event.Event) {
	pl := evt.Payload().(directory.TagsSaveSucceeded)
	pl.TagSet.Notify(evt)
	if v.hasCurrentTagSetChanged(pl.TagSet) {
		return
	}
	u.Skip(v.state.StatusLabel().Set("Saved"))
}

func (v *tagsViewmodelImpl) handleSaveFailed(evt event.Event) {
	pl := evt.Payload().(directory.TagsSaveFailed)
	pl.TagSet.Notify(evt)
	if v.hasCurrentTagSetChanged(pl.TagSet) {
		return
	}
	u.Skip(v.state.StatusLabel().Set("Save failed"))
}

func (v *tagsViewmodelImpl) hasCurrentTagSetChanged(ts *directory.TagSet) bool {
	return v.state.CurrentTagSet().ObjectPath() != ts.ObjectPath()
}
