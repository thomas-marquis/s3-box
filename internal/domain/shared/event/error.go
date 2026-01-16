package event

type BaseErrorEvent struct {
	BaseEvent
	err error
}

func NewBaseErrorEvent(eventType Type, err error) BaseErrorEvent {
	return BaseErrorEvent{BaseEvent: BaseEvent{eventType: eventType}, err: err}
}

func (e BaseErrorEvent) Error() error {
	return e.err
}
