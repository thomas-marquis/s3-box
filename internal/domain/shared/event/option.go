package event

import "context"

type Option func(e BaseEvent) BaseEvent

func WithContext(ctx context.Context) Option {
	return func(e BaseEvent) BaseEvent {
		return BaseEvent{
			eventType: e.eventType,
			ref:       e.ref,
			ctx:       ctx,
		}
	}
}

func WithRef(ref string) Option {
	return func(e BaseEvent) BaseEvent {
		return BaseEvent{
			eventType: e.eventType,
			ctx:       e.ctx,
			ref:       ref,
		}
	}
}
