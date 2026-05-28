package event

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	CarrierTypePrefix = "__carrier__"

	defaultCarrierTimeout     = 60 * time.Second
	defaultCarrierConcurrency = 10
)

func (t Type) IsCarrier() bool {
	return strings.HasPrefix(t.String(), CarrierTypePrefix)
}

type Carrier interface {
	Payload

	// Dispatch dispatches all events in the carrier to the given channel.
	// Depending on bus implementation, this may be blocking or non-blocking.
	Dispatch(bus Bus)
}

type CarriesAll struct {
	Carried          []Event
	DoneEventFactory func(received []Event) Event
	OnTimeout        Event
	DoneCondition    func(sent, received Event) bool //TODO: use a matcher instead???
	maxConcurrency   int
	timeout          time.Duration
}

var (
	_ Carrier = (*CarriesAll)(nil)
)

type carrierConfig struct {
	maxConcurrency int
	timeout        time.Duration
	doneCondition  DoneCondition
}

type CarrierOption func(config *carrierConfig)
type DoneCondition func(sent, received Event) bool

func WithTimeout(d time.Duration) CarrierOption {
	return func(c *carrierConfig) {
		c.timeout = d
	}
}

func WithMaxConcurrency(n int) CarrierOption {
	return func(c *carrierConfig) {
		c.maxConcurrency = n
	}
}

func WithDoneCondition(cond DoneCondition) CarrierOption {
	return func(c *carrierConfig) {
		c.doneCondition = cond
	}
}

// DoneWhenFollowupReceived is a done condition that always returns true.
// This function must not be used in another context than a Carrier.
// Default done function.
func DoneWhenFollowupReceived(sent, received Event) bool {
	return true // always true by construction: we already now that the received event is a followup of the sent one
}

// NewCarriesAll creates a new Carrier that will dispatch all events in the given slice to the event Bus.
// All carried events must have unique Ref (that means they must not be followup from each other), otherwise the behavior is undefined.
// This event carrier has a blocking Dispatch method.
func NewCarriesAll(carried []Event, doneEventFactory func(received []Event) Event, onTimeout Event, opts ...CarrierOption) Event {
	var uniqueRefset = make(map[string]struct{})
	for _, evt := range carried {
		if _, exists := uniqueRefset[evt.Ref]; exists {
			log.Printf("duplicate event ref: %s, undefined behaviour mey will append", evt.Ref)
			continue
		}
		uniqueRefset[evt.Ref] = struct{}{}
	}

	c := &CarriesAll{
		Carried:          carried,
		DoneEventFactory: doneEventFactory,
		OnTimeout:        onTimeout,
	}

	cfg := &carrierConfig{
		maxConcurrency: defaultCarrierConcurrency,
		timeout:        defaultCarrierTimeout,
		doneCondition:  DoneWhenFollowupReceived,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	c.maxConcurrency = cfg.maxConcurrency
	c.timeout = cfg.timeout
	c.DoneCondition = cfg.doneCondition

	return New(c)
}

func (c *CarriesAll) Dispatch(bus Bus) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	evtProcessed := make(map[string]bool)
	evtByRef := make(map[string]Event)
	receivedEvents := make([]Event, 0, len(c.Carried))
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
		On(IsFollowupOf(c.Carried...), func(received Event) {
			mu.Lock()
			if processed, ok := evtProcessed[received.Ref]; ok &&
				!processed &&
				c.DoneCondition(evtByRef[received.Ref], received) {
				evtProcessed[received.Ref] = true
				receivedEvents = append(receivedEvents, received)
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
	t := time.NewTicker(10 * time.Millisecond) // polling may not be the better option...
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			bus.Publish(c.OnTimeout)
			return
		case <-t.C:
			if len(evtProcessed) == len(c.Carried) && allEventsHasBeenProcessed(evtProcessed) {
				bus.Publish(c.DoneEventFactory(receivedEvents))
				return
			}
		}
	}
}

func (c *CarriesAll) Type() Type {
	return Type(CarrierTypePrefix + ".all")
}

//func getUniqueEventTypes(events []Event) []Type {
//	typeSet := make(map[Type]struct{})
//	for _, evt := range events {
//		typeSet[evt.Type()] = struct{}{}
//	}
//	uniques := make([]Type, 0, len(typeSet))
//	for t := range typeSet {
//		uniques = append(uniques, t)
//	}
//	return uniques
//}

func allEventsHasBeenProcessed(eventMap map[string]bool) bool {
	for _, processed := range eventMap {
		if !processed {
			return false
		}
	}
	return true
}

type CarriesSequence struct {
	Carried          []Event
	DoneEventFactory func(received []Event) Event
	OnTimeout        Event
	DoneCondition    DoneCondition

	timeout time.Duration
}

func NewCarriesSequence(carried []Event, doneEventFactory func(received []Event) Event, onTimeout Event, opts ...CarrierOption) Event {
	c := &CarriesSequence{
		Carried:          carried,
		DoneEventFactory: doneEventFactory,
		OnTimeout:        onTimeout,
	}

	cfg := &carrierConfig{
		timeout:       defaultCarrierTimeout,
		doneCondition: DoneWhenFollowupReceived,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	c.timeout = cfg.timeout
	c.DoneCondition = cfg.doneCondition

	return New(c)
}

func (c *CarriesSequence) Type() Type {
	return Type(CarrierTypePrefix + ".sequence")
}

func (c *CarriesSequence) Dispatch(bus Bus) {
	if len(c.Carried) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	go func() {
		defer cancel()
		receivedEvents := c.doDispatch(ctx, bus)
		bus.Publish(c.DoneEventFactory(receivedEvents))
	}()
}

func (c *CarriesSequence) doDispatch(ctx context.Context, bus Bus) (receivedEvents []Event) {
	workload := make(chan Event, 1)
	defer close(workload)

	var currIdx int
	workload <- c.Carried[currIdx]

	var mu sync.Mutex

	for {
		select {
		case evt := <-workload:
			finished := make(chan struct{})
			sub := bus.Subscribe().
				On(IsFollowupOf(evt), func(received Event) {
					if c.DoneCondition(evt, received) {
						mu.Lock()
						defer mu.Unlock()
						receivedEvents = append(receivedEvents, received)
						close(finished)
					}
				})
			sub.ListenWithWorkers(1)
			bus.Publish(evt)

			select {
			case <-finished:
				currIdx++
				if currIdx == len(c.Carried) {
					sub.Detach()
					return
				}
				workload <- c.Carried[currIdx]
			case <-ctx.Done():
				bus.Publish(c.OnTimeout)
				sub.Detach()
				return
			}

			sub.Detach()
		case <-ctx.Done():
			return
		}
	}
}
