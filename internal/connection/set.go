package connection

import "github.com/google/uuid"

type Set struct {
	connections []*Connection
}

type SetOption func(*Set)

func WithConnections(connections []*Connection) SetOption {
	return func(s *Set) {
		s.connections = connections
	}
}

func NewSet(opts ...SetOption) *Set {
	s := &Set{
		connections: make([]*Connection, 0),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Set) Create(
	name, accessKey, secretKey, bucket string,
	options ...ConnectionOption,
) *Connection {
	conn := New(name, accessKey, secretKey, bucket, options...)
	s.connections = append(s.connections, conn)
	return conn
}

func (s *Set) Update(ID uuid.UUID, options ...ConnectionOption) error {
	for i, existingConn := range s.connections {
		if existingConn.ID() == ID {
			s.connections[i].Update(options...)
			return nil
		}
	}
	return ErrConnectionNotFound
}

func (s *Set) Delete(connID uuid.UUID) error {
	for i, conn := range s.connections {
		if conn.ID() == connID {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			return nil
		}
	}
	return ErrConnectionNotFound
}

func (s *Set) Connections() []*Connection {
	return s.connections
}

func (s *Set) Select(connID uuid.UUID) error {
	found := false
	idxToSelect := -1
	for i, conn := range s.connections {
		if conn.Selected() {
			idxToSelect = i
		}
		if conn.ID() == connID {
			idxToSelect = i
			found = true
		}
		conn.Unselect()
	}
	if !found {
		if idxToSelect > -1 {
			s.connections[idxToSelect].Select()
		}
		return ErrConnectionNotFound
	}

	s.connections[idxToSelect].Select()

	return nil
}

func (s *Set) Selected() *Connection {
	for _, conn := range s.connections {
		if conn.Selected() {
			return conn
		}
	}
	return nil
}
