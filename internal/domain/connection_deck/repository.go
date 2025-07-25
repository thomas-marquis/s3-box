package connection_deck

import "context"

type Repository interface {
	Get(ctx context.Context) (*Deck, error)
	Export(ctx context.Context) ([]byte, error)
}
