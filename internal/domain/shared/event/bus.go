package event

type Bus interface {
	Publish(evt Event)
	Subscribe(eventTypes ...Type) <-chan Event
}
