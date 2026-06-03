package event

import "time"

// TestHarnessOption is a function that configures a TestHarness.
type TestHarnessOption func(*TestHarness)

// WithTickDuration sets the duration of a single tick in the marble diagrams.
func WithTickDuration(d time.Duration) TestHarnessOption {
	return func(h *TestHarness) {
		h.tickDuration = d
	}
}

// WithHarnessTimeout sets the total time the harness will wait for expectations to be met.
func WithHarnessTimeout(d time.Duration) TestHarnessOption {
	return func(h *TestHarness) {
		h.timeout = d
	}
}

// WithClock sets the clock used by the harness.
func WithClock(c Clock) TestHarnessOption {
	return func(h *TestHarness) {
		h.clock = c
	}
}
