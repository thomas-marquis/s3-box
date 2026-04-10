package event

type Notifier interface {
	Notify(Event)
}

type noopNotifier struct{}

func (n noopNotifier) Notify(_ Event) {}
