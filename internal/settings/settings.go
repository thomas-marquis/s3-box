package settings

import "errors"

type Settings struct {
	TimeoutInSeconds int
}

var ErrInvalidTimeout = errors.New("timeout must be positive")

func NewSettings(timeoutInSeconds int) (Settings, error) {
	if timeoutInSeconds <= 0 {
		return Settings{}, ErrInvalidTimeout
	}

	return Settings{
		TimeoutInSeconds: timeoutInSeconds,
	}, nil
}
