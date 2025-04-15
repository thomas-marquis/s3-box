package explorer

import (
	"context"
)

type S3FileRepository interface {
	// GetContent retrieves the content of a file by its ID
	GetContent(ctx context.Context, id S3FileID) ([]byte, error)

	// DownloadFile downloads a file to the local filesystem
	DownloadFile(ctx context.Context, key, dest string) error

	// UploadFile uploads a local file to S3
	UploadFile(ctx context.Context, local *LocalFile, remote *S3File) error

	// DeleteFile deletes a file from S3
	DeleteFile(ctx context.Context, id S3FileID) error
}
