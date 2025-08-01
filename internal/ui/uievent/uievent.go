package uievent

import "github.com/thomas-marquis/s3-box/internal/domain/connection_deck"

type UiEventType string

const (
	SelectConnectionType        UiEventType = "uievent.Connection.select"
	SelectConnectionSuccessType UiEventType = "uievent.Connection.select.success"
	SelectConnectionFailureType UiEventType = "uievent.Connection.select.failure"

	CreateConnectionType        UiEventType = "uievent.Connection.create"
	CreateConnectionSuccessType UiEventType = "uievent.Connection.create.success"
	CreateConnectionFailureType UiEventType = "uievent.Connection.create.failure"

	DeleteConnectionType        UiEventType = "uievent.Connection.delete"
	DeleteConnectionSuccessType UiEventType = "uievent.Connection.delete.success"
	DeleteConnectionFailureType UiEventType = "uievent.Connection.delete.failure"
)

type UiEvent interface {
	Type() UiEventType
}

type SelectConnection struct {
	Connection *connection_deck.Connection
}

func (e *SelectConnection) Type() UiEventType {
	return SelectConnectionType
}

type SelectConnectionSuccess struct {
	Connection *connection_deck.Connection
}

func (e *SelectConnectionSuccess) Type() UiEventType {
	return SelectConnectionSuccessType
}

type SelectConnectionFailure struct {
	Error error
}

func (e *SelectConnectionFailure) Type() UiEventType {
	return SelectConnectionFailureType
}

type CreateConnection struct {
	Name      string
	AccessKey string
	SecretKey string
	Bucket    string
	Options   []connection_deck.ConnectionOption
}

func (e *CreateConnection) Type() UiEventType {
	return CreateConnectionType
}

type CreateConnectionSuccess struct {
	Connection *connection_deck.Connection
}

func (e *CreateConnectionSuccess) Type() UiEventType {
	return CreateConnectionSuccessType
}

type CreateConnectionFailure struct {
	Error error
}

func (e *CreateConnectionFailure) Type() UiEventType {
	return CreateConnectionFailureType
}

type DeleteConnection struct {
	Connection *connection_deck.Connection
}

func (e *DeleteConnection) Type() UiEventType {
	return DeleteConnectionType
}

type DeleteConnectionSuccess struct {
	Connection *connection_deck.Connection
}

func (e *DeleteConnectionSuccess) Type() UiEventType {
	return DeleteConnectionSuccessType
}

type DeleteConnectionFailure struct {
	Error error
}

func (e *DeleteConnectionFailure) Type() UiEventType {
	return DeleteConnectionFailureType
}
