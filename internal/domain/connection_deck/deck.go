package connection_deck

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

// Deck represents a collection of connections and maintains a currently selected connection by its ID.
// There is only one deck per user. The deck ensures the consistency of all operations performed over connections.
type Deck struct {
	connections []*Connection
	selectedID  ConnectionID
}

func New(opts ...Option) *Deck {
	d := &Deck{
		connections: make([]*Connection, 0),
		selectedID:  nilConnectionID,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// New creates a new connection with the specified name, access key, secret key, bucket, and optional connection settings.
// The new connection is added to the deck before to be returned.
func (d *Deck) New(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) event.Event {
	conn := newConnection(name, accessKey, secretKey, bucket, options...)
	d.connections = append(d.connections, conn)
	return event.New(CreateConnectionTriggered{
		ConnectionPayload: ConnectionPayload{Conn: conn},
		Deck:              d,
	})
}

// Get returns all the connections currently stored in the deck.
func (d *Deck) Get() []*Connection {
	return d.connections
}

// GetByID searches for a connection by its ID and returns it if found;
// otherwise, returns an ErrNotFound error.
func (d *Deck) GetByID(id ConnectionID) (*Connection, error) {
	for _, conn := range d.connections {
		if id.Is(conn) {
			return conn, nil
		}
	}
	return nil, ErrNotFound
}

// Select sets the provided connection ID as the selected connection in the deck.
// Returns ErrNotFound if the connection ID does not exist in the deck.
func (d *Deck) Select(connID ConnectionID) (event.Event, error) {
	for i, conn := range d.connections {
		if connID.Is(conn) {
			previous, _ := d.GetByID(d.selectedID)
			d.selectedID = d.connections[i].ID()
			return event.New(SelectConnectionTriggered{
				ConnectionPayload: ConnectionPayload{conn},
				Deck:              d,
				Previous:          previous,
			}), nil
		}
	}

	return event.Event{}, ErrNotFound
}

// RemoveAConnection removes a connection with the given ID from the deck.
// If the connection is the currently selected one, the selection is reset.
// Returns ErrNotFound if the connection ID does not exist in the deck.
func (d *Deck) RemoveAConnection(connID ConnectionID) (event.Event, error) {
	for i, conn := range d.connections {
		if connID.Is(conn) {
			d.connections = append(d.connections[:i], d.connections[i+1:]...)
			isSelected := d.selectedID.Is(conn)
			if isSelected {
				d.selectedID = nilConnectionID // Reset selected ID if removed
			}
			return event.New(RemoveConnectionTriggered{
				ConnectionPayload: ConnectionPayload{Conn: conn},
				Deck:              d,
				RemovedIndex:      i,
				WasSelected:       isSelected,
			}), nil
		}
	}

	return event.Event{}, ErrNotFound
}

// SelectedConnection returns the currently selected connection or nil if no connection is selected.
func (d *Deck) SelectedConnection() *Connection {
	if d.selectedID == nilConnectionID {
		return nil
	}
	for _, conn := range d.connections {
		if d.selectedID.Is(conn) {
			return conn
		}
	}
	return nil
}

func (d *Deck) Update(connID ConnectionID, options ...ConnectionOption) (event.Event, error) {
	found := false
	var connIdx int
	var previous Connection
	for i, conn := range d.connections {
		if connID.Is(conn) {
			found = true
			connIdx = i
			previous = *conn
			break
		}
	}
	if !found {
		return event.Event{}, ErrNotFound
	}

	for _, opt := range options {
		opt(d.connections[connIdx])
		d.connections[connIdx].revision++
	}

	return event.New(UpdateConnectionTriggered{
		ConnectionPayload: ConnectionPayload{Conn: d.connections[connIdx]},
		Deck:              d,
		Previous:          &previous,
	}), nil
}

func (d *Deck) Notify(evt event.Event) {
	switch evt.Type() {
	case CreateConnectionFailedType:
		pl := evt.Payload.(CreateConnectionFailed)
		for i, c := range d.connections {
			if c.Is(pl.Connection()) {
				d.connections = append(d.connections[:i], d.connections[i+1:]...)
				return
			}
		}

	case SelectConnectionFailedType:
		pl := evt.Payload.(SelectConnectionFailed)
		prev := pl.Connection()
		if prev != nil {
			d.selectedID = prev.ID()
		}

	case RemoveConnectionFailedType:
		pl := evt.Payload.(RemoveConnectionFailed)
		conn := pl.Connection()
		if conn != nil {
			index := pl.RemovedIndex
			if index < 0 {
				index = 0
			}
			if index > len(d.connections) {
				index = len(d.connections)
			}
			d.connections = append(
				d.connections[:index],
				append(
					[]*Connection{conn},
					d.connections[index:]...,
				)...,
			)
			if pl.WasSelected {
				d.selectedID = conn.ID()
			}
		}

	case UpdateConnectionFailedType:
		pl := evt.Payload.(UpdateConnectionFailed)
		previous := pl.Connection()
		if previous == nil {
			return
		}

		for i, conn := range d.connections {
			if previous.ID().Is(conn) {
				d.connections[i] = previous
				return
			}
		}
	}
}
