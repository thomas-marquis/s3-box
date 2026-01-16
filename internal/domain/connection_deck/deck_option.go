package connection_deck

type Option func(*Deck)

func WithConnections(connections []*Connection) Option {
	return func(s *Deck) {
		s.connections = connections
	}
}
