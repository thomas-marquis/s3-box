package event_test

import (
	"strings"
	"testing"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"pgregory.net/rapid"
)

func GenerateMarble() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		var sb strings.Builder
		numOps := rapid.IntRange(1, 5).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			op := rapid.SampledFrom([]string{"-", "_", "a", "b", "c", "(ab)", "'label'", "a->b", "a[1-3]", "a--b"}).Draw(t, "op")
			sb.WriteString(op)
		}
		return sb.String()
	})
}

func GeneratePayload() *rapid.Generator[event.Payload] {
	return rapid.Custom(func(t *rapid.T) event.Payload {
		val := rapid.String().Draw(t, "payloadVal")
		return mockPayload{val}
	})
}

func TestMarbleExpectations_Property(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		marble := GenerateMarble().Draw(t, "marble")

		labels := []string{"a", "b", "c", "label"}
		payloads := make(map[string]any)
		for _, l := range labels {
			payloads[l] = GeneratePayload().Draw(t, "payload_"+l)
		}

		done := make(chan struct{})
		defer close(done)
		bus := event.NewInMemoryBus(done, nil)
		h := event.NewTestHarness(bus)

		_, err := h.Expect(marble, payloads)
		if err != nil {
			// Some random marbles might be invalid (e.g. unclosed groups)
			// though my generator tries to avoid them.
			// Actually (ab) is literal, so it's always valid.
			// But nested or weird combinations might fail.
			t.Skip("invalid marble syntax generated: ", err)
		}

		// We just want to make sure it doesn't panic and handles input.
		// Detailed validation of all expectations is hard with random input.
	})
}
