package connection_deck

// Deck represents a collection of connections and maintains a currently selected connection by its ID.
// There is only one deck per user. The deck ensure the consistency of all operations performed over connections.
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
) *Connection {
	conn := newConnection(name, accessKey, secretKey, bucket, options...)
	d.connections = append(d.connections, conn)
	return conn
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
func (d *Deck) Select(connID ConnectionID) error {
	found := false
	for i, conn := range d.connections {
		if connID.Is(conn) {
			d.selectedID = d.connections[i].ID()
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}

	return nil
}

// RemoveAConnection removes a connection with the given ID from the deck.
// If the connection is the currently selected one, the selection is reset.
// Returns ErrNotFound if the connection ID does not exist in the deck.
func (d *Deck) RemoveAConnection(connID ConnectionID) error {
	for i, conn := range d.connections {
		if connID.Is(conn) {
			d.connections = append(d.connections[:i], d.connections[i+1:]...)
			if d.selectedID.Is(conn) {
				d.selectedID = nilConnectionID // Reset selected ID if removed
			}
			return nil
		}
	}
	return ErrNotFound
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
