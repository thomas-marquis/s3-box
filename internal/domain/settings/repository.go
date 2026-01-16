package settings

import "context"

type Repository interface {
	// Save saves the settings
	Save(ctx context.Context, settings Settings) error

	// Get returns the settings
	Get(ctx context.Context) (Settings, error)
}
