package explorer

import "context"

type S3DirectoryRepository interface {
	// GetByID retrieves a directory by its ID
	// An error explorer.ErrObjectNotFound is returned if the directory doesn't exist
	// or an other error if a technical problem occurs
	GetByID(ctx context.Context, id S3DirectoryID) (*S3Directory, error)

	// Save saves a directory to the repository: it'll create it if it doesn't exist, or update it if it does
	// An error is returned if a technical problem occurs
	Save(ctx context.Context, d *S3Directory) error
}
