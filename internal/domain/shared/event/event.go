package event

import (
	"context"

	"github.com/google/uuid"
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
	Ref() string
}

type BaseEvent struct {
	ctx context.Context
	ref string
}

func NewBaseEvent(opts ...Option) BaseEvent {
	e := BaseEvent{
		ctx: nil,
		ref: uuid.New().String(),
	}

	for _, opt := range opts {
		e = opt(e)
	}

	if e.ctx == nil {
		e.ctx = context.Background()
	}

	return e
}

func (e BaseEvent) Ref() string {
	return e.ref
}

func (e BaseEvent) Context() context.Context {
	return e.ctx
}

func (e BaseEvent) NewBaseFailure(err error) BaseFailureEvent {
	be := NewBaseFailureEvent(e.Type().AsFailure(), err)
	be.ref = e.ref
	return be
}

func (e BaseEvent) NewBaseSuccess(opts ...Option) BaseEvent {
	bfe := NewBaseEvent(e.Type().AsSuccess(), opts...)
	bfe.ref = e.ref
	return bfe
}
