package event

import (
	"reflect"
	"sync"
	"time"
)

type FollowupManager struct {
	mu         sync.RWMutex
	rules      map[string][]string // trigger -> responses
	activeFrom map[string]int      // response -> activation tick
	triggered  map[string]bool
	bus        Bus
	matcher    *EventMatcher
	done       chan struct{}
}

func NewFollowupManager(bus Bus, matcher *EventMatcher) *FollowupManager {
	return &FollowupManager{
		rules:      make(map[string][]string),
		activeFrom: make(map[string]int),
		triggered:  make(map[string]bool),
		bus:        bus,
		matcher:    matcher,
		done:       make(chan struct{}),
	}
}

func (fm *FollowupManager) AddRule(trigger, response string, activeFrom int) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.rules[trigger] = append(fm.rules[trigger], response)
	fm.activeFrom[response] = activeFrom
}

func (fm *FollowupManager) Trigger(triggerLabel string, evt Event, currentTick int) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	responses, ok := fm.rules[triggerLabel]
	if !ok {
		return
	}
	for _, respLabel := range responses {
		if !fm.triggered[respLabel] && fm.isActiveInternal(respLabel, currentTick) {
			fm.triggered[respLabel] = true
			if payload, ok := fm.matcher.payloads[respLabel]; ok {
				go fm.bus.Publish(NewFollowup(evt, payload))
			}
		}
	}
}

func (fm *FollowupManager) isActive(label string, currentTick int) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.isActiveInternal(label, currentTick)
}

func (fm *FollowupManager) isActiveInternal(label string, currentTick int) bool {
	activationTick, ok := fm.activeFrom[label]
	if !ok {
		return true // Active from T0 by default
	}
	return currentTick >= activationTick
}

func (fm *FollowupManager) StartRetroactiveCheck(clock Clock, startTime time.Time, tickDuration time.Duration, getReceived func() []receivedEvent) {
	go func() {
		ticker := clock.After(tickDuration)
		for {
			select {
			case <-ticker:
				currentTick := int(clock.Now().Sub(startTime) / tickDuration)
				fm.checkRetroactiveFollowups(currentTick, getReceived())
				ticker = clock.After(tickDuration)
			case <-fm.done:
				return
			}
		}
	}()
}

func (fm *FollowupManager) checkRetroactiveFollowups(currentTick int, received []receivedEvent) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	for triggerLabel, responses := range fm.rules {
		for _, respLabel := range responses {
			if fm.triggered[respLabel] {
				continue
			}
			if !fm.isActiveInternal(respLabel, currentTick) {
				continue
			}
			// Check if trigger was received before or at currentTick
			triggerPayload, ok := fm.matcher.payloads[triggerLabel]
			if !ok {
				continue
			}
			for _, rec := range received {
				if rec.tick <= currentTick &&
					rec.evt.Type() == triggerPayload.Type() &&
					reflect.DeepEqual(rec.evt.Payload, triggerPayload) {
					fm.triggered[respLabel] = true
					if respPayload, ok := fm.matcher.payloads[respLabel]; ok {
						go fm.bus.Publish(NewFollowup(rec.evt, respPayload))
					}
					break
				}
			}
		}
	}
}

func (fm *FollowupManager) Stop() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	select {
	case <-fm.done:
	default:
		close(fm.done)
	}
}
