package event

import "context"

type Option func(e BaseEvent) BaseEvent

func WithContext(ctx context.Context) Option {
	return func(e BaseEvent) BaseEvent {
		return BaseEvent{
			eventType:   e.eventType,
			withContext: withContext{ctx: ctx},
		}
	}
}
