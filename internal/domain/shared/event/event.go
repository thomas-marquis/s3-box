package event

import (
	"context"
)

type Type string

func (t Type) String() string {
	return string(t)
}

func (t Type) AsFailure() Type {
	return Type(t.String() + ".failure")
}

func (t Type) AsSuccess() Type {
	return Type(t.String() + ".success")
}

type Event interface {
	Type() Type
	Context() context.Context
}

type withContext struct {
	ctx context.Context
}

func (e withContext) Context() context.Context {
	return e.ctx
}

type BaseEvent struct {
	withContext
	eventType Type
}

func NewBaseEvent(eventType Type, opts ...Option) BaseEvent {
	e := BaseEvent{
		eventType:   eventType,
		withContext: withContext{ctx: nil},
	}

	for _, opt := range opts {
		e = opt(e)
	}

	return e
}

func (e BaseEvent) Type() Type {
	return e.eventType
}
