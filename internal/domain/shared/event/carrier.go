package event

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	CarrierTypePrefix = "__carrier__"

	defaultCarrierTimeout = 60 * time.Second
)

func (t Type) IsCarrier() bool {
	return strings.HasPrefix(t.String(), CarrierTypePrefix)
}

type Carrier interface {
	Payload

	// Dispatch dispatches all events in the carrier to the given channel.
	// This method is not supposed to be blocking.
	Dispatch(bus Bus)
}

type CarriesAll struct {
	Carried        []Event
	OnDone         Event
	OnTimeout      Event
	DoneCondition  func(sent, received Event) bool //TODO: use a matcher instead???
	maxConcurrency int
	timeout        time.Duration
}

var (
	_ Carrier = (*CarriesAll)(nil)
)

type CarrierOption func(*CarriesAll)

func WithTimeout(d time.Duration) CarrierOption {
	return func(c *CarriesAll) {
		c.timeout = d
	}
}

func WithMaxConcurrency(n int) CarrierOption {
	return func(c *CarriesAll) {
		c.maxConcurrency = n
	}
}

func NewCarriesAll(carried []Event, doneCond func(sent, received Event) bool, onDone, onTimeout Event, opts ...CarrierOption) *CarriesAll {
	c := &CarriesAll{Carried: carried, OnDone: onDone, OnTimeout: onTimeout, DoneCondition: doneCond, maxConcurrency: 10, timeout: defaultCarrierTimeout}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CarriesAll) Dispatch(bus Bus) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	evtProcessed := make(map[string]bool)
	evtByRef := make(map[string]Event)
	for _, evt := range c.Carried {
		evtByRef[evt.Ref] = evt
	}
	var mu sync.Mutex

	workload := make(chan Event)
	for range c.maxConcurrency {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case evt, ok := <-workload:
					if !ok {
						return
					}
					mu.Lock()
					evtProcessed[evt.Ref] = false
					mu.Unlock()
					bus.Publish(evt) //TODO; won't prevent to overwhelming the event bus
				}
			}
		}()
	}

	sub := bus.Subscribe().
		On(IsOneOf(getUniqueEventTypes(c.Carried)...), func(received Event) { // TODO: followup events haven't necessary the same types as the carried ones
			mu.Lock()
			if processed, ok := evtProcessed[received.Ref]; ok && !processed && c.DoneCondition(evtByRef[received.Ref], received) { //TODO: use a matcher instead???
				evtProcessed[received.Ref] = true
			}
			mu.Unlock()
		})
	sub.ListenWithWorkers(1)
	defer sub.Detach()

	for _, evt := range c.Carried {
		select {
		case <-ctx.Done():
			break
		case workload <- evt:
		}
	}
	close(workload)

	// Wait for completion or timeout
	t := time.NewTicker(100 * time.Millisecond) // polling may not be the better option...
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			bus.Publish(c.OnTimeout)
			return
		case <-t.C:
			if len(evtProcessed) == len(c.Carried) && allEventsHasBeenProcessed(evtProcessed) {
				bus.Publish(c.OnDone)
				return
			}
		}
	}
}

func (c *CarriesAll) Type() Type {
	return Type(CarrierTypePrefix + ".all")
}

func getUniqueEventTypes(events []Event) []Type {
	typeSet := make(map[Type]struct{})
	for _, evt := range events {
		typeSet[evt.Type()] = struct{}{}
	}
	uniques := make([]Type, 0, len(typeSet))
	for t := range typeSet {
		uniques = append(uniques, t)
	}
	return uniques
}

func allEventsHasBeenProcessed(eventMap map[string]bool) bool {
	for _, processed := range eventMap {
		if !processed {
			return false
		}
	}
	return true
}
