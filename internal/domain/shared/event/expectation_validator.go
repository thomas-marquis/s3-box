package event

import (
	"fmt"
	"strings"
	"sync"
)

type ExpectationValidator struct {
	mu                  sync.RWMutex
	expectations        []timelineEntry
	received            []receivedEvent
	receivedByTick      map[int][]receivedEvent
	matcher             *EventMatcher
	temporalConstraints []MarbleEntry
}

func (v *ExpectationValidator) AddExpectation(tick int, label string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.expectations = append(v.expectations, timelineEntry{tick: tick, label: label})
}

func (v *ExpectationValidator) AddReceived(tick int, evt Event) {
	v.mu.Lock()
	defer v.mu.Unlock()
	rec := receivedEvent{tick: tick, evt: evt}
	v.received = append(v.received, rec)
	if v.receivedByTick == nil {
		v.receivedByTick = make(map[int][]receivedEvent)
	}
	v.receivedByTick[tick] = append(v.receivedByTick[tick], rec)
}

func (v *ExpectationValidator) AddTemporalConstraint(constraint MarbleEntry) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.temporalConstraints = append(v.temporalConstraints, constraint)
}

func (v *ExpectationValidator) ValidateTemporalConstraints() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	// Check ordering, windows, offsets
	// For ordering (a->b):
	for _, constraint := range v.temporalConstraints {
		if constraint.Kind == MarbleEntryOrdering {
			aTick := v.getTickForLabelInternal(constraint.Label)
			bTick := v.getTickForLabelInternal(constraint.Before)
			// If either is missing, we can't satisfy it yet (or ever if finished)
			if aTick == -1 || bTick == -1 || aTick >= bTick {
				return false
			}
		}
	}
	// For windows (a[1-3]):
	for _, constraint := range v.temporalConstraints {
		if constraint.Kind == MarbleEntryWindow {
			tick := v.getTickForLabelInternal(constraint.Label)
			if tick == -1 || tick < constraint.WindowStart || tick > constraint.WindowEnd {
				return false
			}
		}
	}
	// For relative time (a--b):
	for _, constraint := range v.temporalConstraints {
		if constraint.Kind == MarbleEntryRelative {
			aTick := v.getTickForLabelInternal(constraint.Label)
			bTick := v.getTickForLabelInternal(constraint.After)
			if aTick == -1 || bTick == -1 || bTick-aTick != constraint.Offset {
				return false
			}
		}
	}
	return true
}

func (v *ExpectationValidator) getTickForLabel(label string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.getTickForLabelInternal(label)
}

func (v *ExpectationValidator) getTickForLabelInternal(label string) int {
	for _, rec := range v.received {
		if v.matcher.Match(rec.evt, label) {
			return rec.tick
		}
	}
	return -1
}

func (v *ExpectationValidator) Failed() bool {
	if !v.ValidateTemporalConstraints() {
		return true
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	receivedUsed := make(map[int][]bool)
	for tick, events := range v.receivedByTick {
		receivedUsed[tick] = make([]bool, len(events))
	}

	var missing []string
	var unexpected []string

	for _, exp := range v.expectations {
		events, ok := v.receivedByTick[exp.tick]
		if !ok {
			missing = append(missing, fmt.Sprintf("expected %s at tick %d", exp.label, exp.tick))
			continue
		}
		found := false
		for i, rec := range events {
			if !receivedUsed[exp.tick][i] && v.matcher.Match(rec.evt, exp.label) {
				found = true
				receivedUsed[exp.tick][i] = true
				break
			}
		}
		if !found {
			missing = append(missing, fmt.Sprintf("expected %s at tick %d", exp.label, exp.tick))
		}
	}

	for tick, events := range v.receivedByTick {
		for i, rec := range events {
			if !receivedUsed[tick][i] {
				unexpected = append(unexpected, fmt.Sprintf("unexpected %s at tick %d", v.matcher.LabelForEvent(rec.evt), rec.tick))
			}
		}
	}

	return len(missing) > 0 || len(unexpected) > 0
}

func (v *ExpectationValidator) GetFailureReason() string {
	if !v.ValidateTemporalConstraints() {
		return "temporal constraints violated"
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	receivedUsed := make(map[int][]bool)
	for tick, events := range v.receivedByTick {
		receivedUsed[tick] = make([]bool, len(events))
	}

	var missing []string
	var unexpected []string

	for _, exp := range v.expectations {
		events, ok := v.receivedByTick[exp.tick]
		if !ok {
			missing = append(missing, fmt.Sprintf("expected %s at tick %d", exp.label, exp.tick))
			continue
		}
		found := false
		for i, rec := range events {
			if !receivedUsed[exp.tick][i] && v.matcher.Match(rec.evt, exp.label) {
				found = true
				receivedUsed[exp.tick][i] = true
				break
			}
		}
		if !found {
			missing = append(missing, fmt.Sprintf("expected %s at tick %d", exp.label, exp.tick))
		}
	}

	for tick, events := range v.receivedByTick {
		for i, rec := range events {
			if !receivedUsed[tick][i] {
				unexpected = append(unexpected, fmt.Sprintf("unexpected %s at tick %d", v.matcher.LabelForEvent(rec.evt), rec.tick))
			}
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
		return msg
	}
	return ""
}

func (v *ExpectationValidator) AllExpectationsMet() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	receivedUsed := make(map[int][]bool)
	for tick, events := range v.receivedByTick {
		receivedUsed[tick] = make([]bool, len(events))
	}

	for _, exp := range v.expectations {
		events, ok := v.receivedByTick[exp.tick]
		if !ok {
			return false
		}
		found := false
		for i, rec := range events {
			if !receivedUsed[exp.tick][i] && v.matcher.Match(rec.evt, exp.label) {
				found = true
				receivedUsed[exp.tick][i] = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (v *ExpectationValidator) GetReceived() []receivedEvent {
	v.mu.RLock()
	defer v.mu.RUnlock()
	res := make([]receivedEvent, len(v.received))
	copy(res, v.received)
	return res
}
