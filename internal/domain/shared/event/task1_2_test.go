package event_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func TestLateFollowupActivation(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	bus := event.NewInMemoryBus(done, nil)
	h := event.NewTestHarness(bus,
		event.WithTickDuration(10*time.Millisecond),
		event.WithHarnessTimeout(1*time.Second),
	)

	evtMap := map[string]any{
		"e1": mockPayload{"e1"},
		"r1": mockPayload{"r1"},
	}

	// r1 activates at tick 5.
	// We expect r1.
	if _, err := h.Expect("--'e1'--'r1'", evtMap); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Given("-----'r1<-e1'", evtMap); err != nil {
		t.Fatal(err)
	}

	// Publish e1 at tick 2.
	// r1 should trigger at tick 5.
	time.AfterFunc(20*time.Millisecond, func() {
		bus.Publish(event.New(mockPayload{"e1"}))
	})

	success, err := h.PrayAndWait()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, success, "r1 should have been triggered retroactively at tick 5")
}
