package connections

import "github.com/google/uuid"

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
