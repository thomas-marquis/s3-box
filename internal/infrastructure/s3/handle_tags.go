package s3

import (
	"fmt"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
)

func (h *EventHandler) handleLoadTags(e event.Event) {
	ctx := e.Context()
	pl := e.Payload().(directory.TagsLoadTriggered)
	tagSet := pl.TagSet

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed loading tags for file %s: %w", pl.TagSet.ObjectPath(), err))
		h.bus.Publish(e.NewFollowup(directory.TagsLoadFailed{
			TagSet: tagSet,
			Error:  err,
		}))
	}

	client, err := h.clientFactory.Get(ctx, pl.TagSet.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	key := mapPathToObjectKey(pl.TagSet.ObjectPath())
	output, err := client.GetObjectTagging(ctx, key)
	if err != nil {
		handleError(fmt.Errorf("failed to get tags for file %s: %w", key, err))
		return
	}

	tags := s3client.MapToDomainTags(output.TagSet)
	h.bus.Publish(e.NewFollowup(directory.TagsLoadSucceeded{
		TagSet: tagSet,
		Tags:   tags,
	}))
}

func (h *EventHandler) handleSaveTags(e event.Event) {
	ctx := e.Context()
	pl := e.Payload().(directory.TagsSaveTriggered)
	tagSet := pl.TagSet
	tags := pl.Tags

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed saving tags for file %s: %w", pl.TagSet.ObjectPath(), err))
		h.bus.Publish(e.NewFollowup(directory.TagsSaveFailed{
			TagSet: tagSet,
			Error:  err,
		}))
	}

	client, err := h.clientFactory.Get(ctx, pl.TagSet.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	key := mapPathToObjectKey(pl.TagSet.ObjectPath())
	s3Tags := s3client.MapFromDomainTags(tags)

	if _, err := client.PutObjectTagging(ctx, key, s3Tags); err != nil {
		handleError(fmt.Errorf("failed to save tags for file %s: %w", key, err))
		return
	}

	h.bus.Publish(e.NewFollowup(directory.TagsSaveSucceeded{
		TagSet: tagSet,
		Tags:   tags,
	}))
}
