package settings

type Repository interface {
	// Save saves the settings
	Save(settings Settings) error

	// Get returns the settings
	Get() (Settings, error)
}