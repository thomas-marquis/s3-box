package event

import (
	"fmt"
	"testing"
)

type benchmarkPayload struct {
	val string
}

func (p benchmarkPayload) Type() Type { return Type("benchmark") }

func BenchmarkFailedInternal(b *testing.B) {
	matcher := NewEventMatcher()
	v := &ExpectationValidator{
		matcher: matcher,
	}

	// Setup: 1000 expectations, 1000 received events
	for i := 0; i < 1000; i++ {
		label := fmt.Sprintf("label%d", i)
		payload := benchmarkPayload{fmt.Sprintf("payload%d", i)}
		matcher.AddPayload(label, payload)
		v.AddExpectation(i, label)
		v.AddReceived(i, New(payload))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Failed()
	}
}
