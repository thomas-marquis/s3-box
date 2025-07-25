package connection_deck

type Deck struct {
	connections []*Connection
	selectedID  ConnectionID
}

func New(opts ...ConnectionsOption) *Deck {
	d := &Deck{
		connections: make([]*Connection, 0),
		selectedID:  nilConnectionID,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Deck) New(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	conn := newConnection(name, accessKey, secretKey, bucket, options...)
	d.connections = append(d.connections, conn)
	return conn
}

func (d *Deck) Get() []*Connection {
	return d.connections
}

func (d *Deck) GetByID(id ConnectionID) (*Connection, error) {
	for _, conn := range d.connections {
		if id.Is(conn) {
			return conn, nil
		}
	}
	return nil, ErrNotFound
}

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
