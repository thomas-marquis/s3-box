package explorer

import (
	"context"
	"github.com/thomas-marquis/s3-box/internal/connection"
)

type Repository interface {
	// ListDirectoryContent returns all directories and files in the given directory
	ListDirectoryContent(ctx context.Context, dir *S3Directory) ([]*S3Directory, []*S3File, error)

	GetFileContent(ctx context.Context, file *S3File) ([]byte, error)

	// SetConnection sets the current connection to be used
	SetConnection(ctx context.Context, c *connection.Connection) error

	DownloadFile(ctx context.Context, file *S3File, dest string) error

	UploadFile(ctx context.Context, local *LocalFile, remote *S3File) error

	DeleteFile(ctx context.Context, remote *S3File) error
}

type S3DirectoryRepository interface {
	Save(ctx context.Context, dir *S3Directory) error

	GetByPath(ctx context.Context, path string) (*S3Directory, error)
}
