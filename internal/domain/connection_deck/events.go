package connection_deck

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

const (
	SelectEventType event.Type = "deck.select"
	RemoveEventType event.Type = "deck.remove"
	CreateEventType event.Type = "deck.create"
	UpdateEventType event.Type = "deck.update"
)

type withDeck struct {
	deck *Deck
}

func (e withDeck) Deck() *Deck {
	return e.deck
}

type withConnection struct {
	connection *Connection
}

func (e withConnection) Connection() *Connection {
	return e.connection
}

type SelectEvent struct {
	event.BaseEvent
	withConnection
	withDeck
	previous *Connection
}

func NewSelectEvent(deck *Deck, selected *Connection, previous *Connection) SelectEvent {
	return SelectEvent{
		event.NewBaseEvent(SelectEventType),
		withConnection{selected},
		withDeck{deck},
		previous,
	}
}

func (e SelectEvent) Previous() *Connection {
	return e.previous
}

type SelectSuccessEvent struct {
	event.BaseEvent
	withConnection
	withDeck
}

func NewSelectSuccessEvent(deck *Deck, connection *Connection) SelectSuccessEvent {
	return SelectSuccessEvent{
		event.NewBaseEvent(SelectEventType.AsSuccess()),
		withConnection{connection},
		withDeck{deck},
	}
}

type SelectFailureEvent struct {
	event.BaseErrorEvent
	withConnection
}

func NewSelectFailureEvent(err error, conn *Connection) SelectFailureEvent {
	return SelectFailureEvent{
		event.NewBaseErrorEvent(SelectEventType.AsFailure(), err),
		withConnection{conn},
	}
}

type RemoveEvent struct {
	event.BaseEvent
	withConnection
	withDeck
	removedIndex int
	wasSelected  bool
}

func NewRemoveEvent(deck *Deck, connection *Connection, index int, wasSelected bool) RemoveEvent {
	return RemoveEvent{
		event.NewBaseEvent(RemoveEventType),
		withConnection{connection},
		withDeck{deck},
		index,
		wasSelected,
	}
}

func (e RemoveEvent) RemovedIndex() int {
	return e.removedIndex
}

func (e RemoveEvent) WasSelected() bool {
	return e.wasSelected
}

type RemoveSuccessEvent struct {
	event.BaseEvent
	withConnection
	withDeck
}

func NewRemoveSuccessEvent(deck *Deck, connection *Connection) RemoveSuccessEvent {
	return RemoveSuccessEvent{
		event.NewBaseEvent(RemoveEventType.AsSuccess()),
		withConnection{connection},
		withDeck{deck},
	}
}

type RemoveFailureEvent struct {
	event.BaseErrorEvent
	withConnection
	removedIndex int
	wasSelected  bool
}

func NewRemoveFailureEvent(err error, index int, wasSelected bool, conn *Connection) RemoveFailureEvent {
	return RemoveFailureEvent{
		event.NewBaseErrorEvent(RemoveEventType.AsFailure(), err),
		withConnection{conn},
		index,
		wasSelected,
	}
}

func (e RemoveFailureEvent) RemovedIndex() int {
	return e.removedIndex
}

func (e RemoveFailureEvent) WasSelected() bool {
	return e.wasSelected
}

type CreateEvent struct {
	event.BaseEvent
	withConnection
	withDeck
}

func NewCreateEvent(deck *Deck, connection *Connection) CreateEvent {
	return CreateEvent{
		event.NewBaseEvent(CreateEventType),
		withConnection{connection},
		withDeck{deck},
	}
}

type CreateSuccessEvent struct {
	event.BaseEvent
	withConnection
	withDeck
}

func NewCreateSuccessEvent(deck *Deck, connection *Connection) CreateSuccessEvent {
	return CreateSuccessEvent{
		event.NewBaseEvent(CreateEventType.AsSuccess()),
		withConnection{connection},
		withDeck{deck},
	}
}

type CreateFailureEvent struct {
	event.BaseErrorEvent
	withConnection
}

func NewCreateFailureEvent(err error, conn *Connection) CreateFailureEvent {
	return CreateFailureEvent{
		event.NewBaseErrorEvent(CreateEventType.AsFailure(), err),
		withConnection{conn},
	}
}

type UpdateEvent struct {
	event.BaseEvent
	withDeck
	withConnection
	previous *Connection
}

func NewUpdateEvent(deck *Deck, previous *Connection, newConnection *Connection) UpdateEvent {
	return UpdateEvent{
		event.NewBaseEvent(UpdateEventType),
		withDeck{deck},
		withConnection{newConnection},
		previous,
	}
}

func (e UpdateEvent) Previous() *Connection {
	return e.previous
}

type UpdateSuccessEvent struct {
	event.BaseEvent
	withDeck
	withConnection
}

func NewUpdateSuccessEvent(deck *Deck, newConnection *Connection) UpdateSuccessEvent {
	return UpdateSuccessEvent{
		event.NewBaseEvent(UpdateEventType.AsSuccess()),
		withDeck{deck},
		withConnection{newConnection},
	}
}

type UpdateFailureEvent struct {
	event.BaseErrorEvent
	withConnection
}

func NewUpdateFailureEvent(err error, previous *Connection) UpdateFailureEvent {
	return UpdateFailureEvent{
		event.NewBaseErrorEvent(UpdateEventType.AsFailure(), err),
		withConnection{previous},
	}
}
