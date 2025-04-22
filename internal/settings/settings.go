package settings

import "errors"

type Settings struct {
	TimeoutInSeconds int
}

var ErrInvalidTimeout = errors.New("timeout must be positive")

const (
	DefaultTimeoutInSeconds = 15
)

func NewSettings(timeoutInSeconds int) (Settings, error) {
	if timeoutInSeconds <= 0 {
		return Settings{}, ErrInvalidTimeout
	}

	return Settings{
		TimeoutInSeconds: timeoutInSeconds,
	}, nil
}

func DefaultSettings() Settings {
	return Settings{
		TimeoutInSeconds: DefaultTimeoutInSeconds,
	}
}
