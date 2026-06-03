package event_test

import (
	"testing"
	"time"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func BenchmarkPrayAndWait(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus, event.WithTickDuration(1*time.Millisecond))

		evtMap := map[string]any{
			"a": mockPayload{"a"},
			"b": mockPayload{"b"},
		}

		_, _ = h.Expect("--a--b--|", evtMap)

		go func() {
			time.Sleep(2 * time.Millisecond)
			bus.Publish(event.New(mockPayload{"a"}))
			time.Sleep(2 * time.Millisecond)
			bus.Publish(event.New(mockPayload{"b"}))
		}()

		_, _ = h.PrayAndWait()
		close(done)
	}
}
