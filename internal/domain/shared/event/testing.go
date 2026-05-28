package event

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

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

// TestHarness is a utility for testing event-driven logic by defining expected event sequences
// and providing automated responses using marble diagram syntax.
type TestHarness struct {
	bus          Bus
	timeout      time.Duration
	tickDuration time.Duration

	payloads     map[string]Payload
	expectations []timelineEntry
	publications []timelineEntry
	followups    map[string][]string // target label -> response labels

	toPublish []Event

	received []receivedEvent
	mu       sync.Mutex
	done     chan struct{}
	started  bool

	expectCalled bool
	givenCalled  bool

	startTime time.Time
	sub       *Subscriber
}

// NewTestHarness creates a new TestHarness instance attached to the provided Bus.
// By default, it uses a 10ms tick duration and a 10s timeout.
func NewTestHarness(bus Bus, opts ...TestHarnessOption) *TestHarness {
	h := &TestHarness{
		bus:          bus,
		timeout:      time.Second * 10,
		tickDuration: 10 * time.Millisecond,
		payloads:     make(map[string]Payload),
		followups:    make(map[string][]string),
		done:         make(chan struct{}),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Publish registers an event to be sent to the underlying bus when PrayAndWait is called.
// This method is lazy.
func (h *TestHarness) Publish(evt Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toPublish = append(h.toPublish, evt)
}

// Expect registers a sequence of events that the harness should wait for.
// A single dash '-' represents a unique time tick when nothing happens.
// A contiguous underscore sequence '___' counts for one time tick.
// Labels can be single characters or quoted strings like 'label'.
// Groups '(ab)' allow multiple events at the same tick.
// The parser panics if the syntax is incorrect.
// This method must be called exactly once.
func (h *TestHarness) Expect(marble string, evtMap map[string]Payload) *TestHarness {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.expectCalled {
		panic("Expect can only be called once")
	}
	h.expectCalled = true

	for k, v := range evtMap {
		h.payloads[k] = v
	}

	ops := parseMarble(marble)
	currentTick := 0
	for _, o := range ops {
		h.applyOp(o, &currentTick, true)
	}

	return h
}

// Given registers a sequence of events to be published or followup rules.
// Timed events follow the marble syntax (ticks and labels).
// Followup rules are specified as 'response<-trigger', e.g., 'r1<-e1'.
// When the trigger event is received, the response event is published as a followup.
// Groups '(ab)' allow multiple events/rules at the same tick.
// This method can be called at most once.
func (h *TestHarness) Given(marble string, evtMap map[string]Payload) *TestHarness {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.givenCalled {
		panic("Given can only be called once")
	}
	h.givenCalled = true

	for k, v := range evtMap {
		h.payloads[k] = v
	}

	ops := parseMarble(marble)
	currentTick := 0
	for _, o := range ops {
		h.applyOp(o, &currentTick, false)
	}

	return h
}

func (h *TestHarness) applyOp(o op, currentTick *int, isExpect bool) {
	switch o.kind {
	case opTick:
		*currentTick++
	case opEvent:
		if isExpect {
			h.expectations = append(h.expectations, timelineEntry{tick: *currentTick, label: o.label})
		} else {
			h.publications = append(h.publications, timelineEntry{tick: *currentTick, label: o.label})
		}
		*currentTick++
	case opFollowup:
		if isExpect {
			panic("followup syntax '<-' is not allowed in Expect diagrams")
		}
		h.followups[o.target] = append(h.followups[o.target], o.label)
		*currentTick++
	case opGroup:
		for _, sub := range o.subOps {
			switch sub.kind {
			case opEvent:
				if isExpect {
					h.expectations = append(h.expectations, timelineEntry{tick: *currentTick, label: sub.label})
				} else {
					h.publications = append(h.publications, timelineEntry{tick: *currentTick, label: sub.label})
				}
			case opFollowup:
				if isExpect {
					panic("followup syntax '<-' is not allowed in Expect diagrams")
				}
				h.followups[sub.target] = append(h.followups[sub.target], sub.label)
			default:
				panic("invalid operation in group")
			}
		}
		*currentTick++
	}
}

// PrayAndWait starts the simulation and blocks until all expected events are received,
// or the timeout is reached. All lazy Publish calls are executed at the start.
// It returns true if all expectations were met, false otherwise.
func (h *TestHarness) PrayAndWait() bool {
	h.mu.Lock()
	if !h.expectCalled {
		h.mu.Unlock()
		panic("Expect must be called before PrayAndWait")
	}

	if h.started {
		h.mu.Unlock()
		return !h.failedInternal()
	}
	h.started = true
	h.startTime = time.Now()

	// Subscribe to all events
	h.sub = h.bus.Subscribe().On(IsAny(), h.handleEvent)
	h.sub.ListenWithWorkers(1)

	// Publish lazy events
	for _, evt := range h.toPublish {
		h.bus.Publish(evt)
	}
	h.toPublish = nil

	// Schedule timed publications
	for _, pub := range h.publications {
		p := pub // capture for closure
		time.AfterFunc(time.Duration(p.tick)*h.tickDuration, func() {
			h.mu.Lock()
			payload, ok := h.payloads[p.label]
			h.mu.Unlock()
			if ok {
				h.bus.Publish(New(payload))
			}
		})
	}

	h.mu.Unlock()

	// Wait for expectations or timeout
	select {
	case <-h.done:
	case <-time.After(h.timeout):
	}

	h.sub.Detach()

	return !h.failedInternal()
}

func (h *TestHarness) failedInternal() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	receivedUsed := make([]bool, len(h.received))

	// Check if all expectations were met
	for _, exp := range h.expectations {
		found := false
		for i, rec := range h.received {
			if !receivedUsed[i] && rec.tick == exp.tick && h.matches(rec.evt, exp.label) {
				found = true
				receivedUsed[i] = true
				break
			}
		}
		if !found {
			return true
		}
	}

	// Check for unexpected events (any received event not consumed by expectations)
	for _, used := range receivedUsed {
		if !used {
			return true
		}
	}

	return false
}

func (h *TestHarness) handleEvent(evt Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tick := int(time.Since(h.startTime) / h.tickDuration)
	h.received = append(h.received, receivedEvent{tick: tick, evt: evt})

	// Check for followups
	for label, payload := range h.payloads {
		if evt.Type() == payload.Type() && reflect.DeepEqual(evt.Payload, payload) {
			if responses, ok := h.followups[label]; ok {
				for _, respLabel := range responses {
					if respPayload, ok := h.payloads[respLabel]; ok {
						go h.bus.Publish(NewFollowup(evt, respPayload))
					}
				}
			}
		}
	}

	// Check if all expectations are met to signal completion
	if h.allExpectationsMetInternal() {
		select {
		case <-h.done:
		default:
			close(h.done)
		}
	}
}

func (h *TestHarness) allExpectationsMetInternal() bool {
	receivedUsed := make([]bool, len(h.received))
	for _, exp := range h.expectations {
		found := false
		for i, rec := range h.received {
			if !receivedUsed[i] && rec.tick == exp.tick && h.matches(rec.evt, exp.label) {
				found = true
				receivedUsed[i] = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// For early exit, we also need to ensure no unexpected events arrived so far
	// However, more events might arrive later. So allExpectationsMetInternal
	// only signals that the MINIMUM requirements are met.
	// But according to the rules, ANY unexpected event makes it fail.
	// So if we have more received events than expectations met, it's already failed?
	// Not necessarily, they might match later expectations.

	// Wait, if an event arrived at T0 but we expected it at T1, it's a failure.
	// Our failedInternal check handles this.

	// To be safe, let's only close h.done if EXACTLY the expected events arrived
	// up to the current count of received events.
	if len(h.received) != len(h.expectations) {
		return false
	}

	return true
}

func (h *TestHarness) matches(evt Event, label string) bool {
	payload, ok := h.payloads[label]
	if !ok {
		return false
	}
	return evt.Type() == payload.Type() && reflect.DeepEqual(evt.Payload, payload)
}

func parseMarble(marble string) []op {
	var ops []op
	i := 0
	for i < len(marble) {
		c := marble[i]
		switch {
		case c == '-':
			ops = append(ops, op{kind: opTick})
			i++
		case c == '_':
			for i < len(marble) && marble[i] == '_' {
				i++
			}
			ops = append(ops, op{kind: opTick})
		case c == ' ':
			i++
		case c == '(':
			i++
			start := i
			for i < len(marble) && marble[i] != ')' {
				i++
			}
			if i >= len(marble) {
				panic("invalid marble: unclosed group '('")
			}
			groupContent := marble[start:i]
			i++
			subOps := parseMarble(groupContent)
			// Filter out ticks from group
			var filtered []op
			for _, so := range subOps {
				if so.kind == opTick {
					panic("ticks '-' or '_' are not allowed inside groups '(ab)'")
				}
				if so.kind == opGroup {
					panic("nested groups are not allowed")
				}
				filtered = append(filtered, so)
			}
			ops = append(ops, op{kind: opGroup, subOps: filtered})
		case c == '\'':
			i++
			start := i
			for i < len(marble) && marble[i] != '\'' {
				i++
			}
			if i >= len(marble) {
				panic("invalid marble: unclosed quote")
			}
			label := marble[start:i]
			i++
			if strings.Contains(label, "<-") {
				parts := strings.Split(label, "<-")
				if len(parts) != 2 {
					panic("invalid followup syntax: " + label)
				}
				ops = append(ops, op{kind: opFollowup, label: parts[0], target: parts[1]})
			} else {
				ops = append(ops, op{kind: opEvent, label: label})
			}
		default:
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				ops = append(ops, op{kind: opEvent, label: string(c)})
				i++
			} else {
				panic(fmt.Sprintf("invalid character in marble at position %d: %c", i, c))
			}
		}
	}
	return ops
}
