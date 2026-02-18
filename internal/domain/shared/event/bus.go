package event

type Bus interface {
	Publish(evt Event)
	PublishV2(evt Event)
	Subscribe() <-chan Event
	SubscribeV2() *Subscriber
}
