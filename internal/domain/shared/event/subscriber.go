package event

type Subscriber struct {
	registered map[Matcher]func(Event)
	events     <-chan Event
	started    bool
}

func NewSubscriber(event <-chan Event) *Subscriber {
	return &Subscriber{registered: make(map[Matcher]func(Event)), events: event}
}

func (s *Subscriber) On(matcher Matcher, callback func(Event)) *Subscriber {
	if s.started {
		panic("cannot register callback after listening started")
	}

	if _, exists := s.registered[matcher]; exists {
		return s
	}

	s.registered[matcher] = callback
	return s
}

func (s *Subscriber) listen() {
	for event := range s.events {
		for matcher, callback := range s.registered {
			if matcher.Match(event) {
				callback(event)
			}
		}
	}
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
		for event := range s.events {
			for matcher, callback := range s.registered {
				if matcher.Match(event) {
					go callback(event)
				}
			}
		}
	}()
}

func (s *Subscriber) Accept(event Event) bool {
	for matcher := range s.registered {
		if matcher.Match(event) {
			return true
		}
	}
	return false
}
