package event

type Bus interface {
	Publish(evt Event)
	Subscribe() <-chan Event
}
