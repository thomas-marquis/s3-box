package event

type BaseFailureEvent struct {
	BaseEvent
	err error
}

func NewBaseFailureEvent(eventType Type, err error, opts ...Option) BaseFailureEvent {
	e := BaseFailureEvent{
		BaseEvent: BaseEvent{eventType: eventType},
		err:       err,
	}

	for _, opt := range opts {
		e.BaseEvent = opt(e.BaseEvent)
	}

	return e
}

func (e BaseFailureEvent) Error() error {
	return e.err
}
