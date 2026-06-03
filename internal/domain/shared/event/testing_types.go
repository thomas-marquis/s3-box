package event

// timelineEntry represents an event or action scheduled at a specific tick.
type timelineEntry struct {
	tick  int
	label string
}

// receivedEvent records an event received by the harness.
type receivedEvent struct {
	tick int
	evt  Event
}
