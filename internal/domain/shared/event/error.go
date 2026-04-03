package event

type FailureEvent interface {
	Error() error
}

type BaseFailureEvent struct {
	BaseEvent
	err error
}

func NewBaseFailureEvent(eventType Type, err error) BaseFailureEvent {
	return BaseFailureEvent{
		BaseEvent: BaseEvent{eventType: eventType},
		err:       err,
	}
}

func (e BaseFailureEvent) Error() error {
	return e.err
}
