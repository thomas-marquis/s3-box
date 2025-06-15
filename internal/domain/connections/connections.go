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
