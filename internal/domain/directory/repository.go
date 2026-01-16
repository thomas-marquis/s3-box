package directory

import (
	"context"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

type Repository interface {
	GetByPath(ctx context.Context, connID connection_deck.ConnectionID, path Path) (*Directory, error)
	GetFileContent(ctx context.Context, connID connection_deck.ConnectionID, file *File) (*Content, error)
}
