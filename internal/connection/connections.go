package connection

import "github.com/google/uuid"

type Connections struct {
	connections []*Connection
}

type ConnectionsOption func(*Connections)

func WithConnections(connections []*Connection) ConnectionsOption {
	return func(c *Connections) {
		c.connections = connections
	}
}

func NewConnections(opts ...ConnectionsOption) *Connections {
	c := &Connections{
		connections: make([]*Connection, 0),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Connections) NewConnection(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	conn := NewConnection(name, accessKey, secretKey, bucket, options...)
	c.connections = append(c.connections, conn)
	return conn
}

func (c *Connections) Delete(connID uuid.UUID) error {
	for i, conn := range c.connections {
		if conn.ID == connID {
			c.connections = append(c.connections[:i], c.connections[i+1:]...)
			return nil
		}
	}
	return ErrConnectionNotFound
}

func (c *Connections) Connections() []*Connection {
	return c.connections
}

func (c *Connections) Select(connID uuid.UUID) error {
	found := false
	for _, conn := range c.connections {
		if conn.ID == connID {
			conn.IsSelected = true
			found = true
		} else if conn.IsSelected {
			conn.IsSelected = false
		}
	}
	if !found {
		return ErrConnectionNotFound
	}
	return nil
}

func (c *Connections) Selected() *Connection {
	for _, conn := range c.connections {
		if conn.IsSelected {
			return conn
		}
	}
	return nil
}

func (c *Connections) Update(conn Connection) error {
	for i, existingConn := range c.connections {
		if existingConn.ID == conn.ID {
			c.connections[i].Update(&conn)
			return nil
		}
	}
	return ErrConnectionNotFound
}
