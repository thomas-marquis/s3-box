package connection_deck

import (
	"io"

	"github.com/thomas-marquis/it-happened/event"
)

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

func (e SelectConnectionTriggered) EventType() event.Type {
	return SelectConnectionTriggeredType
}

type SelectConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e SelectConnectionSucceeded) EventType() event.Type {
	return SelectConnectionSucceededType
}

type SelectConnectionFailed struct {
	Err error
	ConnectionPayload
}

func (e SelectConnectionFailed) EventType() event.Type {
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

func (e RemoveConnectionTriggered) EventType() event.Type {
	return RemoveConnectionTriggeredType
}

type RemoveConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e RemoveConnectionSucceeded) EventType() event.Type {
	return RemoveConnectionSucceededType
}

type RemoveConnectionFailed struct {
	ConnectionPayload
	Err          error
	RemovedIndex int
	WasSelected  bool
}

func (e RemoveConnectionFailed) EventType() event.Type {
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

func (e CreateConnectionTriggered) EventType() event.Type {
	return CreateConnectionTriggeredType
}

type CreateConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (e CreateConnectionSucceeded) EventType() event.Type {
	return CreateConnectionSucceededType
}

type CreateConnectionFailed struct {
	ConnectionPayload
	Err error
}

func (e CreateConnectionFailed) EventType() event.Type {
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

func (UpdateConnectionTriggered) EventType() event.Type {
	return UpdateConnectionTriggeredType
}

type UpdateConnectionSucceeded struct {
	ConnectionPayload
	Deck *Deck
}

func (UpdateConnectionSucceeded) EventType() event.Type {
	return UpdateConnectionSucceededType
}

type UpdateConnectionFailed struct {
	ConnectionPayload
	Err error
}

func (UpdateConnectionFailed) EventType() event.Type {
	return UpdateConnectionFailedType
}

func (e UpdateConnectionFailed) Error() error {
	return e.Err
}

const (
	ImportTriggeredType         event.Type = "deck.connection.import.triggered"
	ImportFailedType            event.Type = "deck.connection.import.failed"
	ImportConfirmationAskedType event.Type = "deck.connection.import.confirmation.asked"
	ImportSucceededType         event.Type = "deck.connection.import.succeeded"
)

type ImportTriggered struct {
	Deck     *Deck
	JSONFile io.ReadCloser
}

func (ImportTriggered) EventType() event.Type {
	return ImportTriggeredType
}

type ImportFailed struct {
	Deck *Deck
	Err  error
}

var (
	_ ErrorGetter = (*ImportFailed)(nil)
)

func (ImportFailed) EventType() event.Type {
	return ImportFailedType
}

func (e ImportFailed) Error() error {
	return e.Err
}

type ImportConfirmationAsked struct {
	Deck           *Deck
	NewConnections []*Connection
}

func (ImportConfirmationAsked) EventType() event.Type {
	return ImportConfirmationAskedType
}

type ImportSucceeded struct {
	Deck           *Deck
	NewConnections []*Connection
}

func (ImportSucceeded) EventType() event.Type {
	return ImportSucceededType
}
