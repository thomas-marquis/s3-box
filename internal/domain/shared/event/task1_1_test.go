package event_test

import (
	"sync"
	"testing"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func TestConcurrentPublish(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	bus := event.NewInMemoryBus(done, nil)
	h, err := event.NewTestHarness(bus).Expect("--a", map[string]any{"a": mockPayload{"test"}})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		h.Publish(event.New(mockPayload{"concurrent"}))
	}()
	go func() {
		defer wg.Done()
		_, _ = h.PrayAndWait()
	}()
	wg.Wait()
}
