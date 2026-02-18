package event

type Bus interface {
	Publish(evt Event)
	Subscribe() *Subscriber
}
