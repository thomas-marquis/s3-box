package connections

type ConnectionsOption func(*Connections)

func WithConnections(connections []*Connection) ConnectionsOption {
	return func(s *Connections) {
		s.connections = connections
	}
}
