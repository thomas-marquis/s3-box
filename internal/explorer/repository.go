package explorer

import (
	"context"
	"github.com/thomas-marquis/s3-box/internal/connection"
)

type Repository interface {
	// ListDirectoryContent returns all directories and files in the given directory
	ListDirectoryContent(ctx context.Context, dir *Directory) ([]*Directory, []*RemoteFile, error)

	GetFileContent(ctx context.Context, file *RemoteFile) ([]byte, error)

	// SetConnection sets the current connection to be used
	SetConnection(ctx context.Context, c *connection.Connection) error

	DownloadFile(ctx context.Context, file *RemoteFile, dest string) error

	UploadFile(ctx context.Context, local *LocalFile, remote *RemoteFile) error
}
