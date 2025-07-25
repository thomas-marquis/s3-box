package connection_deck

type ConnectionsOption func(*Deck)

func WithConnections(connections []*Connection) ConnectionsOption {
	return func(s *Deck) {
		s.connections = connections
	}
}
