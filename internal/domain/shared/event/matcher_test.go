package event_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type fakePayload2 struct{}

func (fakePayload2) Type() event.Type {
	return "fake.payload.2"
}

func TestIsFollowupOf(t *testing.T) {
	t.Run("should match when event is a followup of at least one of the given events", func(t *testing.T) {
		// Given
		events := []event.Event{
			event.New(fakePayload("event1")),
			event.New(fakePayload("event2")),
		}

		m := event.IsFollowupOf(events...)

		incoming := event.NewFollowup(events[0], fakePayload2{})

		// When
		res := m.Match(incoming)

		// Then
		assert.True(t, res)
	})

	t.Run("should not match when event is not a followup of at least one of the given events", func(t *testing.T) {
		// Given
		events := []event.Event{
			event.New(fakePayload("event1")),
			event.New(fakePayload("event2")),
		}

		m := event.IsFollowupOf(events...)

		incoming := event.New(fakePayload2{})

		// When
		res := m.Match(incoming)

		// Then
		assert.False(t, res)
	})

	t.Run("should not match the same event as the given ones", func(t *testing.T) {
		e := event.New(fakePayload("event2"))
		events := []event.Event{
			event.New(fakePayload("event1")),
			e,
		}

		m := event.IsFollowupOf(events...)

		// When
		res := m.Match(e)

		// Then
		assert.False(t, res)
	})
}
