package event

type Bus interface {
	PublishV2(evt Event)
	SubscribeV2() *Subscriber
}
