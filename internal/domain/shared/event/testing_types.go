package event

// opKind defines the type of operation parsed from a marble string.
type opKind int

const (
	opTick opKind = iota
	opEvent
	opFollowup
	opGroup
)

// op represents a single operation in a marble sequence.
type op struct {
	kind   opKind
	label  string
	target string
	subOps []op // for opGroup
}

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
