package directory

import (
	"context"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

type Repository interface {
	GetByPath(ctx context.Context, connID connection_deck.ConnectionID, path Path) (*Directory, error)
	DownloadFile(ctx context.Context, connID connection_deck.ConnectionID, file *File, destPath string) error
	LoadContent(ctx context.Context, connID connection_deck.ConnectionID, file *File) ([]byte, error)
}
