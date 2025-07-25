package directory

type EventPublisher struct {
	subscribers map[chan Event]struct{}
}

func NewEventPublisher() *EventPublisher {
	return &EventPublisher{
		subscribers: make(map[chan Event]struct{}),
	}
}

func (e *EventPublisher) Subscribe(subscriber chan Event) {
	e.subscribers[subscriber] = struct{}{}
}

func (e *EventPublisher) Unsubscribe(subscriber chan Event) {
	delete(e.subscribers, subscriber)
}

func (e *EventPublisher) Publish(event Event) {
	go func() {
		for subscriber := range e.subscribers {
			subscriber <- event
		}
	}()
}
