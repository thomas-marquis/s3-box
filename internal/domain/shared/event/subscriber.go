package event

import "sync"

type Subscriber struct {
	sync.RWMutex

	registered map[Matcher]func(Event)
	events     chan Event
	started    bool
	bus        Bus
	done       chan struct{}
}

func NewSubscriber(event chan Event) *Subscriber {
	return &Subscriber{
		registered: make(map[Matcher]func(Event)),
		events:     event,
		bus:        NewInMemoryBus(make(<-chan struct{}), nil), // TODO: remove this
		done:       make(chan struct{}),
	}
}

func NewSubscriberWithBus(event chan Event, bus Bus) *Subscriber {
	return &Subscriber{registered: make(map[Matcher]func(Event)), events: event, bus: bus, done: make(chan struct{})}
}

func (s *Subscriber) On(matcher Matcher, callback func(Event)) *Subscriber {
	if s.started {
		panic("cannot register callback after listening started")
	}

	s.Lock()
	defer s.Unlock()
	if _, exists := s.registered[matcher]; exists {
		return s
	}

	s.registered[matcher] = callback
	return s
}

func (s *Subscriber) listen() {
	for {
		select {
		case <-s.done:
			return
		case event := <-s.events:
			if event.Type().IsCarrier() {
				c := event.Payload.(Carrier)
				c.Dispatch(s.bus)
				continue
			}
			s.RLock()
			for matcher, callback := range s.registered {
				s.RUnlock()
				if matcher.Match(event) {
					callback(event)
				}
				s.RLock()
			}
			s.RUnlock()
		}
	}

	//for event := range s.events {
	//	if event.Type().IsCarrier() {
	//		c := event.Payload.(Carrier)
	//		c.Dispatch(s.bus)
	//		continue
	//	}
	//	s.RLock()
	//	for matcher, callback := range s.registered {
	//		s.RUnlock()
	//		if matcher.Match(event) {
	//			callback(event)
	//		}
	//		s.RLock()
	//	}
	//	s.RUnlock()
	//}
}

func (s *Subscriber) ListenWithWorkers(workers int) {
	s.started = true
	for i := 0; i < workers; i++ {
		go s.listen()
	}
}

func (s *Subscriber) ListenNonBlocking() {
	s.started = true
	go func() {

		for {
			select {
			case <-s.done:
				return
			case event := <-s.events:
				if event.Type().IsCarrier() {
					c := event.Payload.(Carrier)
					c.Dispatch(s.bus)
					continue
				}
				s.RLock()
				for matcher, callback := range s.registered {
					s.RUnlock()
					if matcher.Match(event) {
						go callback(event)
					}
					s.RLock()
				}
				s.RUnlock()
			}
		}

		//for event := range s.events {
		//	if event.Type().IsCarrier() {
		//		c := event.Payload.(Carrier)
		//		c.Dispatch(s.events)
		//		continue
		//	}
		//	s.RLock()
		//	for matcher, callback := range s.registered {
		//		s.RUnlock()
		//		if matcher.Match(event) {
		//			go callback(event)
		//		}
		//		s.RLock()
		//	}
		//	s.RUnlock()
		//}
	}()
}

func (s *Subscriber) Accept(event Event) bool {
	s.RLock()
	defer s.RUnlock()
	for matcher := range s.registered {
		if matcher.Match(event) {
			return true
		}
	}
	return false
}

func (s *Subscriber) Detach() {
	close(s.done)
}
