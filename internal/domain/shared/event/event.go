package event

import (
	"context"

	"github.com/google/uuid"
)

type Type string

func (t Type) String() string {
	return string(t)
}

type Payload interface {
	Type() Type
}

type Event struct {
	Payload   Payload
	Context   context.Context
	Ref       string
	eventType Type
}

func (e Event) Type() Type {
	return e.eventType
}

func New(payload Payload, opts ...Option) Event {
	e := Event{
		Ref:       uuid.New().String(),
		Payload:   payload,
		eventType: payload.Type(),
	}

	for _, opt := range opts {
		e = opt(e)
	}

	if e.Context == nil {
		e.Context = context.Background()
	}

	return e
}

func NewFollowup(previous Event, newPayload Payload, opts ...Option) Event {
	prevRef := previous.Ref
	if prevRef == "" {
		prevRef = uuid.New().String()
	}
	ne := New(newPayload, opts...)
	ne.Ref = prevRef
	return ne
}
