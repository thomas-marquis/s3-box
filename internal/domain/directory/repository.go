package directory

import (
	"context"

	"github.com/thomas-marquis/s3-box/internal/domain/connections"
)

type Repository interface {
	GetByPath(ctx context.Context, connID connections.ConnectionID, path Path) (*Directory, error)
	DownloadFile(ctx context.Context, connID connections.ConnectionID, file *File, destPath string) error
	LoadContent(ctx context.Context, connID connections.ConnectionID, file *File) ([]byte, error)
}
