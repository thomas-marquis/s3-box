package connections

type Connections struct {
	connections []*Connection
	selectedID  ConnectionID
}

func New(opts ...ConnectionsOption) *Connections {
	conns := &Connections{
		connections: make([]*Connection, 0),
		selectedID:  nilConnectionID,
	}
	for _, opt := range opts {
		opt(conns)
	}
	return conns
}

func (c *Connections) NewConnection(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	conn := newConnection(name, accessKey, secretKey, bucket, options...)
	c.connections = append(c.connections, conn)
	return conn
}

func (c *Connections) Get() []*Connection {
	return c.connections
}

func (c *Connections) Select(connID ConnectionID) error {
	found := false
	for i, conn := range c.connections {
		if connID.Is(conn) {
			c.selectedID = c.connections[i].ID()
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}

	return nil
}

func (c *Connections) RemoveAConnection(connID ConnectionID) error {
	for i, conn := range c.connections {
		if connID.Is(conn) {
			c.connections = append(c.connections[:i], c.connections[i+1:]...)
			if c.selectedID.Is(conn) {
				c.selectedID = nilConnectionID // Reset selected ID if removed
			}
			return nil
		}
	}
	return ErrNotFound
}

// SelectedConnection returns the currently selected connection or nil if no connection is selected.
func (c *Connections) SelectedConnection() *Connection {
	if c.selectedID == nilConnectionID {
		return nil
	}
	for _, conn := range c.connections {
		if c.selectedID.Is(conn) {
			return conn
		}
	}
	return nil
}
