package event

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// TestHarness is a utility for testing event-driven logic by defining expected event sequences
// and providing automated responses using marble diagram syntax.
type TestHarness struct {
	bus          Bus
	timeout      time.Duration
	tickDuration time.Duration

	payloads      map[string]Payload
	expectations  []timelineEntry
	publications  []timelineEntry
	followups     map[string][]string // target label -> response labels
	followupTicks map[string]int      // response label -> tick when rule becomes active

	toPublish []Event

	received  []receivedEvent
	triggered map[string]bool // track which followups have been triggered to avoid duplicates
	mu        sync.Mutex
	done      chan struct{}
	started   bool

	expectCalled bool
	givenCalled  bool

	startTime  time.Time
	sub        *Subscriber
	failReason string
}

// NewTestHarness creates a new TestHarness instance attached to the provided Bus.
// By default, it uses a 10ms tick duration and a 10s timeout.
func NewTestHarness(bus Bus, opts ...TestHarnessOption) *TestHarness {
	h := &TestHarness{
		bus:           bus,
		timeout:       time.Second * 10,
		tickDuration:  10 * time.Millisecond,
		payloads:      make(map[string]Payload),
		followups:     make(map[string][]string),
		followupTicks: make(map[string]int),
		triggered:     make(map[string]bool),
		done:          make(chan struct{}),
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
		h.followupTicks[o.label] = *currentTick
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
				h.followupTicks[sub.label] = *currentTick
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
		failed := h.failedInternal()
		h.mu.Unlock()
		return !failed
	}
	h.started = true
	h.startTime = time.Now()
	h.failReason = ""
	h.triggered = make(map[string]bool)

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

	// Start a ticker to check for retroactive followups at each tick
	// This handles the case where a followup rule becomes active after its trigger was already received
	h.startRetroactiveCheck()

	// Wait for expectations or timeout
	select {
	case <-h.done:
	case <-time.After(h.timeout):
		h.mu.Lock()
		if h.failReason == "" {
			h.failReason = "timeout: not all expectations met within the timeout period"
		}
		h.mu.Unlock()
	}

	h.sub.Detach()

	return !h.failedInternal()
}

// startRetroactiveCheck starts a goroutine that checks at each tick if any followup rules
// have become active and their triggers were already received
func (h *TestHarness) startRetroactiveCheck() {
	// Find max tick we need to check
	maxTick := 0
	for _, entry := range h.publications {
		if entry.tick > maxTick {
			maxTick = entry.tick
		}
	}
	for _, activationTick := range h.followupTicks {
		if activationTick > maxTick {
			maxTick = activationTick
		}
	}
	maxTick += 5 // Extra buffer for done event

	go func() {
		ticker := time.NewTicker(h.tickDuration)
		defer ticker.Stop()
		currentTick := 1
		for {
			select {
			case <-ticker.C:
				h.mu.Lock()
				h.checkRetroactiveFollowups(currentTick)
				h.mu.Unlock()
				currentTick++
				if currentTick > maxTick {
					return
				}
			case <-h.done:
				return
			}
		}
	}()
}

// checkRetroactiveFollowups checks if any followup rules became active at currentTick
// and triggers them if their trigger events were already received
func (h *TestHarness) checkRetroactiveFollowups(currentTick int) {
	for triggerLabel, responseLabels := range h.followups {
		for _, respLabel := range responseLabels {
			// Check if this response becomes active at currentTick and hasn't been triggered yet
			if activationTick, ok := h.followupTicks[respLabel]; ok && activationTick == currentTick && !h.triggered[respLabel] {
				// This followup rule just became active
				// Check if the trigger was already received before currentTick
				triggerPayload, ok := h.payloads[triggerLabel]
				if !ok {
					continue
				}
				for _, rec := range h.received {
					if rec.tick < currentTick &&
						rec.evt.Type() == triggerPayload.Type() &&
						reflect.DeepEqual(rec.evt.Payload, triggerPayload) {
						// Trigger was received earlier, trigger the followup now
						if respPayload, ok := h.payloads[respLabel]; ok {
							go h.bus.Publish(NewFollowup(rec.evt, respPayload))
							h.triggered[respLabel] = true
						}
						break // Move to next followup rule after finding a match
					}
				}
			}
		}
	}
}

// GetFailureReason returns a detailed error message explaining why PrayAndWait returned false.
// This should be called after PrayAndWait returns false to get diagnostic information.
func (h *TestHarness) GetFailureReason() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.failReason
}

func (h *TestHarness) failedInternal() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	receivedUsed := make([]bool, len(h.received))
	var missing []string
	var unexpected []string

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
			missing = append(missing, fmt.Sprintf("expected %s at tick %d", exp.label, exp.tick))
		}
	}

	// Check for unexpected events (any received event not consumed by expectations)
	for i, rec := range h.received {
		if !receivedUsed[i] {
			unexpected = append(unexpected, fmt.Sprintf("unexpected %s at tick %d", h.labelForEvent(rec.evt), rec.tick))
		}
	}

	if len(missing) > 0 || len(unexpected) > 0 {
		var msg string
		if len(missing) > 0 {
			msg = "Missing: " + strings.Join(missing, ", ")
		}
		if len(unexpected) > 0 {
			if msg != "" {
				msg += "; "
			}
			msg += "Unexpected: " + strings.Join(unexpected, ", ")
		}
		h.failReason = msg
		return true
	}

	return false
}

func (h *TestHarness) labelForEvent(evt Event) string {
	for label, payload := range h.payloads {
		if evt.Type() == payload.Type() && reflect.DeepEqual(evt.Payload, payload) {
			return label
		}
	}
	return fmt.Sprintf("%v", evt.Payload)
}

func (h *TestHarness) handleEvent(evt Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tick := int(time.Since(h.startTime) / h.tickDuration)
	h.received = append(h.received, receivedEvent{tick: tick, evt: evt})

	// Check for followups - only trigger if the followup rule is active at this tick
	for label, payload := range h.payloads {
		if evt.Type() == payload.Type() && reflect.DeepEqual(evt.Payload, payload) {
			if responses, ok := h.followups[label]; ok {
				for _, respLabel := range responses {
					if h.isFollowupActive(respLabel, tick) && !h.triggered[respLabel] {
						h.triggered[respLabel] = true
						if respPayload, ok := h.payloads[respLabel]; ok {
							go h.bus.Publish(NewFollowup(evt, respPayload))
						}
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

func (h *TestHarness) isFollowupActive(label string, currentTick int) bool {
	activationTick, ok := h.followupTicks[label]
	if !ok {
		// If no specific activation tick, followup is active from T0
		return true
	}
	return currentTick >= activationTick
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
