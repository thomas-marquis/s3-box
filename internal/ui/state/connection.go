package state

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/u"
)

type ConnectionState struct {
	connections binding.List[*connection_deck.Connection]
	deck        *connection_deck.Deck
}

func newConnectionState() *ConnectionState {
	return &ConnectionState{
		connections: binding.NewList[*connection_deck.Connection](connection_deck.Compare),
	}
}

func (s *ConnectionState) Init(deck *connection_deck.Deck) {
	s.deck = deck
	for _, c := range deck.Get() {
		u.Skip(s.connections.Append(c))
	}
}

func (s *ConnectionState) List() binding.List[*connection_deck.Connection] {
	return s.connections
}

func (s *ConnectionState) Deck() *connection_deck.Deck {
	return s.deck
}

func (s *ConnectionState) FindOrNil(id connection_deck.ConnectionID) *connection_deck.Connection {
	connections, err := s.connections.Get()
	if err != nil {
		return nil
	}

	for _, conn := range connections {
		if conn.ID() == id {
			return conn
		}
	}
	return nil
}
