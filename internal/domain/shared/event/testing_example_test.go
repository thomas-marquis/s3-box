package event_test

import (
	"fmt"
	"time"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func ExampleTestHarness() {
	done := make(chan struct{})
	defer close(done)
	bus := event.NewInMemoryBus(done, nil)
	h := event.NewTestHarness(bus, event.WithTickDuration(100*time.Millisecond))

	payloadA := mockPayload{"a"}
	payloadB := mockPayload{"b"}

	// Expect a then b after 1 tick
	_, _ = h.Expect("-a-b", map[string]any{
		"a": payloadA,
		"b": payloadB,
	})

	// Publish a then b after 1 tick
	_, _ = h.Given("-a-b", map[string]any{
		"a": payloadA,
		"b": payloadB,
	})

	success, _ := h.PrayAndWait()
	if !success {
		fmt.Println(h.GetFailureReason())
	}
	fmt.Printf("Success: %v\n", success)
	// Output: Success: true
}
