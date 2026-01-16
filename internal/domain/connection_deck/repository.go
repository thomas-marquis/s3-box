package connection_deck

import (
	"context"
	"io"
)

type Repository interface {
	Get(ctx context.Context) (*Deck, error)
	Export(ctx context.Context, file io.Writer) error
}
