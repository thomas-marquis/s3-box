package event

import "context"

type Option func(e Event) Event

func WithContext(ctx context.Context) Option {
	return func(e Event) Event {
		e.Context = ctx
		return e
	}
}

func WithRef(ref string) Option {
	return func(e Event) Event {
		e.Ref = ref
		return e
	}
}
