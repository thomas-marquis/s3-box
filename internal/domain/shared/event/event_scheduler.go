package event

import (
	"time"
)

type ScheduledEvent struct {
	Tick    int
	Payload Payload
}

type EventScheduler struct {
	bus          Bus
	tickDur      time.Duration
	publications []ScheduledEvent
	timers       []Timer
}

func NewEventScheduler(bus Bus) *EventScheduler {
	return &EventScheduler{
		bus:     bus,
		tickDur: 10 * time.Millisecond,
	}
}

func (s *EventScheduler) Schedule(payload Payload, tick int) {
	s.publications = append(s.publications, ScheduledEvent{Tick: tick, Payload: payload})
}

func (s *EventScheduler) Start(clock Clock) {
	for _, pub := range s.publications {
		timer := clock.AfterFunc(time.Duration(pub.Tick)*s.tickDur, func() {
			s.bus.Publish(New(pub.Payload))
		})
		s.timers = append(s.timers, timer)
	}
}

func (s *EventScheduler) Stop() {
	for _, timer := range s.timers {
		timer.Stop()
	}
	s.timers = nil
}
