package event

type Notifier interface {
	Notify(Event)
}

type NopNotifier struct{}

func (n NopNotifier) Notify(_ Event) {}
