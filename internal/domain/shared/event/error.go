package event

type ErrorEvent interface {
	Error() error
}

type BaseErrorEvent struct {
	BaseEvent
	err error
}

var _ ErrorEvent = (*BaseErrorEvent)(nil)

func NewBaseErrorEvent(eventType Type, err error) BaseErrorEvent {
	return BaseErrorEvent{BaseEvent: BaseEvent{eventType: eventType}, err: err}
}

func (e BaseErrorEvent) Error() error {
	return e.err
}
