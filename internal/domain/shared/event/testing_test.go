package event_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type mockPayload struct {
	val string
}

func (m mockPayload) Type() event.Type { return event.Type("mockPayload") }

func TestTestHarness(t *testing.T) {
	t.Run("should succeed with ticks and lazy publish", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus, event.WithTickDuration(10*time.Millisecond))

		evtMap := map[string]event.Payload{
			"a": mockPayload{"a"},
			"b": mockPayload{"b"},
		}

		h.Expect("a-b", evtMap)
		h.Publish(event.New(mockPayload{"a"}))
		h.Given("--b", evtMap)

		assert.True(t, h.PrayAndWait(), "harness should have succeeded")
	})

	t.Run("should handle underscore sequences as one tick", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus, event.WithTickDuration(5*time.Millisecond))

		evtMap := map[string]event.Payload{
			"a": mockPayload{"a"},
		}

		h.Expect("____-_a", evtMap)
		h.Given("---a", evtMap)

		assert.True(t, h.PrayAndWait(), "harness should have succeeded with underscores")
	})

	t.Run("should handle explicit followups (same tick)", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus, event.WithTickDuration(10*time.Millisecond))

		evtMap := map[string]event.Payload{
			"q": mockPayload{"query"},
			"r": mockPayload{"response"},
		}

		h.Expect("(qr)", evtMap).
			Given("'r<-q'", evtMap)

		h.Publish(event.New(mockPayload{"query"}))

		assert.True(t, h.PrayAndWait(), "harness should have succeeded with explicit followup in group")
	})

	t.Run("should fail if unexpected event arrives", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus,
			event.WithTickDuration(5*time.Millisecond),
			event.WithHarnessTimeout(100*time.Millisecond),
		)

		evtMap := map[string]event.Payload{
			"a": mockPayload{"a"},
			"x": mockPayload{"unexpected"},
		}

		h.Expect("-a", evtMap)
		h.Publish(event.New(mockPayload{"x"}))
		h.Given("-a", evtMap)

		assert.False(t, h.PrayAndWait(), "harness should have failed due to unexpected event at T0")
	})

	t.Run("should support simultaneous events in Expect", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus, event.WithTickDuration(10*time.Millisecond))

		evtMap := map[string]event.Payload{
			"a": mockPayload{"a"},
			"b": mockPayload{"b"},
			"c": mockPayload{"c"},
		}

		h.Expect("-(ab)c", evtMap)

		// Publish a and b at T1
		time.AfterFunc(10*time.Millisecond, func() {
			bus.Publish(event.New(mockPayload{"a"}))
			bus.Publish(event.New(mockPayload{"b"}))
		})
		// Publish c at T2
		time.AfterFunc(20*time.Millisecond, func() {
			bus.Publish(event.New(mockPayload{"c"}))
		})

		assert.True(t, h.PrayAndWait(), "harness should handle simultaneous events in Expect")
	})

	t.Run("should support simultaneous events in Given", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus, event.WithTickDuration(10*time.Millisecond))

		evtMap := map[string]event.Payload{
			"a": mockPayload{"a"},
			"b": mockPayload{"b"},
		}

		h.Expect("(ab)", evtMap)
		h.Given("(ab)", evtMap)

		assert.True(t, h.PrayAndWait(), "harness should handle simultaneous events in Given")
	})

	t.Run("should panic if Expect is not called", func(t *testing.T) {
		bus := event.NewInMemoryBus(make(chan struct{}), nil)
		h := event.NewTestHarness(bus)

		assert.Panics(t, func() {
			h.PrayAndWait()
		})
	})

	t.Run("should panic if Expect is called twice", func(t *testing.T) {
		bus := event.NewInMemoryBus(make(chan struct{}), nil)
		h := event.NewTestHarness(bus)

		h.Expect("a", nil)
		assert.Panics(t, func() {
			h.Expect("b", nil)
		})
	})

	t.Run("should panic if Given is called twice", func(t *testing.T) {
		bus := event.NewInMemoryBus(make(chan struct{}), nil)
		h := event.NewTestHarness(bus)

		h.Given("a", nil)
		assert.Panics(t, func() {
			h.Given("b", nil)
		})
	})

	t.Run("should panic on invalid syntax", func(t *testing.T) {
		bus := event.NewInMemoryBus(make(chan struct{}), nil)
		h := event.NewTestHarness(bus)

		assert.Panics(t, func() {
			h.Expect("!", nil)
		})

		assert.Panics(t, func() {
			h.Expect("'unclosed", nil)
		})

		assert.Panics(t, func() {
			h.Expect("(a-b)", nil) // ticks not allowed in groups
		})
	})
}
