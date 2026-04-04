package connection_deck

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

type ConnectionGetter interface {
	Connection() *Connection
}

type ConnectionPayload struct {
	Conn *Connection
}

type ErrorGetter interface {
	Error() error
}

func (e ConnectionPayload) Connection() *Connection {
	return e.Conn
}

const (
	SelectConnectionTriggeredType event.Type = "deck.connection.select.triggered"
	SelectConnectionSucceededType event.Type = "deck.connection.select.succeeded"
	SelectConnectionFailedType    event.Type = "deck.connection.select.failed"
)

var (
	_ ConnectionGetter = (*SelectConnectionTriggered)(nil)
	_ ConnectionGetter = (*SelectConnectionSucceeded)(nil)
	_ ConnectionGetter = (*SelectConnectionFailed)(nil)
	_ ErrorGetter      = (*SelectConnectionFailed)(nil)
)

type SelectConnectionTriggered struct {
	ConnectionPayload
	Deck     *Deck
	Previous *Connection
}

func (e SelectConnectionTriggered) Type() event.Type {
	return SelectConnectionTriggeredType
}

type SelectConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e SelectConnectionSucceeded) Type() event.Type {
	return SelectConnectionSucceededType
}

type SelectConnectionFailed struct {
	Err error
	ConnectionPayload
}

func (e SelectConnectionFailed) Type() event.Type {
	return SelectConnectionFailedType
}

func (e SelectConnectionFailed) Error() error {
	return e.Err
}

const (
	RemoveConnectionTriggeredType event.Type = "deck.connection.remove.triggered"
	RemoveConnectionSucceededType event.Type = "deck.connection.remove.succeeded"
	RemoveConnectionFailedType    event.Type = "deck.connection.remove.failed"
)

var (
	_ ConnectionGetter = (*RemoveConnectionTriggered)(nil)
	_ ConnectionGetter = (*RemoveConnectionSucceeded)(nil)
	_ ConnectionGetter = (*RemoveConnectionFailed)(nil)
	_ ErrorGetter      = (*RemoveConnectionFailed)(nil)
)

type RemoveConnectionTriggered struct {
	ConnectionPayload
	Deck         *Deck
	RemovedIndex int
	WasSelected  bool
}

func (e RemoveConnectionTriggered) Type() event.Type {
	return RemoveConnectionTriggeredType
}

type RemoveConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e RemoveConnectionSucceeded) Type() event.Type {
	return RemoveConnectionSucceededType
}

type RemoveConnectionFailed struct {
	ConnectionPayload
	Err          error
	RemovedIndex int
	WasSelected  bool
}

func (e RemoveConnectionFailed) Type() event.Type {
	return RemoveConnectionFailedType
}

func (e RemoveConnectionFailed) Error() error {
	return e.Err
}

const (
	CreateConnectionTriggeredType event.Type = "deck.connection.create.triggered"
	CreateConnectionSucceededType event.Type = "deck.connection.create.succeeded"
	CreateConnectionFailedType    event.Type = "deck.connection.create.failed"
)

var (
	_ ConnectionGetter = (*CreateConnectionTriggered)(nil)
	_ ConnectionGetter = (*CreateConnectionSucceeded)(nil)
	_ ConnectionGetter = (*CreateConnectionFailed)(nil)
	_ ErrorGetter      = (*CreateConnectionFailed)(nil)
)

type CreateConnectionTriggered struct {
	ConnectionPayload
	Deck *Deck
}

func (e CreateConnectionTriggered) Type() event.Type {
	return CreateConnectionTriggeredType
}

type CreateConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e CreateConnectionSucceeded) Type() event.Type {
	return CreateConnectionSucceededType
}

type CreateConnectionFailed struct {
	ConnectionPayload
	Err error
}

func (e CreateConnectionFailed) Type() event.Type {
	return CreateConnectionFailedType
}

func (e CreateConnectionFailed) Error() error {
	return e.Err
}

const (
	UpdateConnectionTriggeredType event.Type = "deck.connection.update.triggered"
	UpdateConnectionSucceededType event.Type = "deck.connection.update.succeeded"
	UpdateConnectionFailedType    event.Type = "deck.connection.update.failed"
)

var (
	_ ConnectionGetter = (*UpdateConnectionTriggered)(nil)
	_ ConnectionGetter = (*UpdateConnectionSucceeded)(nil)
	_ ConnectionGetter = (*UpdateConnectionFailed)(nil)
	_ ErrorGetter      = (*UpdateConnectionFailed)(nil)
)

type UpdateConnectionTriggered struct {
	ConnectionPayload
	Deck     *Deck
	Previous *Connection
}

func (e UpdateConnectionTriggered) Type() event.Type {
	return UpdateConnectionTriggeredType
}

type UpdateConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e UpdateConnectionSucceeded) Type() event.Type {
	return UpdateConnectionSucceededType
}

type UpdateConnectionFailed struct {
	ConnectionPayload
	Err error
}

func (e UpdateConnectionFailed) Type() event.Type {
	return UpdateConnectionFailedType
}

func (e UpdateConnectionFailed) Error() error {
	return e.Err
}
