package settings

import "errors"

type Settings struct {
	TimeoutInSeconds        int
	MaxFilePreviewSizeBytes int
}

var ErrInvalidTimeout = errors.New("timeout must be positive")

const (
	DefaultTimeoutInSeconds        = 15
	DefaultMaxFilePreviewSizeBytes = 1024 * 10
)

func NewSettings(timeoutInSeconds int, maxFilePreviewSizeBytes int) (Settings, error) {
	if timeoutInSeconds <= 0 {
		return Settings{}, ErrInvalidTimeout
	}

	if maxFilePreviewSizeBytes == 0 {
		maxFilePreviewSizeBytes = DefaultMaxFilePreviewSizeBytes
	}

	return Settings{
		TimeoutInSeconds:        timeoutInSeconds,
		MaxFilePreviewSizeBytes: maxFilePreviewSizeBytes,
	}, nil
}

func DefaultSettings() Settings {
	return Settings{
		TimeoutInSeconds:        DefaultTimeoutInSeconds,
		MaxFilePreviewSizeBytes: DefaultMaxFilePreviewSizeBytes,
	}
}
