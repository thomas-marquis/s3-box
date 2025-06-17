package directory

import (
	"context"

	"github.com/thomas-marquis/s3-box/internal/domain/connections"
)

type Repository interface {
	GetByPath(ctx context.Context, connID connections.ConnectionID, path Path) (*Directory, error)
	Save(ctx context.Context, connID connections.ConnectionID, dir *Directory) error
	Delete(ctx context.Context, connID connections.ConnectionID, Path Path) error
	DownloadFile(ctx context.Context, connID connections.ConnectionID, file *File, destPath string) error
	UploadFile(ctx context.Context, connID connections.ConnectionID, srcPath string, destFile *File) error
	LoadContent(ctx context.Context, connID connections.ConnectionID, file *File) ([]byte, error)
}
