package state

import (
	"errors"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/u"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
)

type ConnectionState struct {
	connections binding.List[*connection_deck.Connection]
	deck        *connection_deck.Deck
	selected    binding.Item[*connection_deck.Connection]
}

func newConnectionState() *ConnectionState {
	return &ConnectionState{
		connections: binding.NewList[*connection_deck.Connection](connection_deck.Compare),
		selected:    binding.NewItem[*connection_deck.Connection](connection_deck.Compare),
	}
}

func (s *ConnectionState) Init(deck *connection_deck.Deck) {
	s.deck = deck
	for _, c := range deck.Get() {
		u.Skip(s.connections.Append(c))
	}

	if selected := deck.SelectedConnection(); selected != nil {
		u.Skip(s.selected.Set(selected))
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

func (s *ConnectionState) IsReadOnly() bool {
	if !s.HasSelected() {
		return false
	}
	return s.deck.SelectedConnection().ReadOnly()
}

func (s *ConnectionState) Selected() binding.Item[*connection_deck.Connection] {
	return s.selected
}

// HasSelected returns true if a connection has been selected.
func (s *ConnectionState) HasSelected() bool {
	return s.deck.SelectedConnection() != nil
}

func (s *ConnectionState) Remove(conn *connection_deck.Connection) error {
	found := false
	allConnections, _ := s.connections.Get()
	for _, prevConn := range allConnections {
		if prevConn.Is(conn) {
			found = s.connections.Remove(prevConn) == nil
		}
	}

	if !found {
		return ErrConnectionNotFound
	}

	return nil

}

func (s *ConnectionState) Update(c *connection_deck.Connection) error {
	found := false
	for i, conn := range s.deck.Get() {
		if conn.Is(c) {
			found = true
			updatedConn := *c // Create a copy to have a new ref in the binding
			if err := s.connections.SetValue(i, &updatedConn); err != nil {
				return err
			}

			// Necessary workaround to trigger the refresh in the UI
			placeholderConn := &connection_deck.Connection{}
			u.Skip(s.connections.Append(placeholderConn))
			u.Skip(s.connections.Remove(placeholderConn))
		}
	}

	if !found {
		return ErrConnectionNotFound
	}

	return nil
}
