package settings

import "errors"

type ColorTheme int

const (
	ColorThemeLight ColorTheme = iota
	ColorThemeDark
	ColorThemeSystem
)

func (c ColorTheme) String() string {
	switch c {
	case ColorThemeLight:
		return "light"
	case ColorThemeDark:
		return "dark"
	case ColorThemeSystem:
		return "system"
	default:
		return "unknown"
	}
}

func NewColorThemeFromString(s string) (ColorTheme, error) {
	switch s {
	case "light":
		return ColorThemeLight, nil
	case "dark":
		return ColorThemeDark, nil
	case "system":
		return ColorThemeSystem, nil
	default:
		return -1, errors.New("invalid color theme")
	}
}

type Settings struct {
	TimeoutInSeconds int
	Color            ColorTheme
	MaxFilePreviewSizeBytes int
}

var ErrInvalidTimeout = errors.New("timeout must be positive")

const (
	DefaultTimeoutInSeconds = 15
	DefaultMaxFilePreviewSizeBytes = 1024 * 1024 * 5 // 5MB
)

func NewSettings(timeoutInSeconds int, maxFilePreviewSizeBytes int) (Settings, error) {
	if timeoutInSeconds <= 0 {
		return Settings{}, ErrInvalidTimeout
	}

	return Settings{
		TimeoutInSeconds: timeoutInSeconds,
		MaxFilePreviewSizeBytes: maxFilePreviewSizeBytes,
	}, nil
}

func DefaultSettings() Settings {
	return Settings{
		TimeoutInSeconds: DefaultTimeoutInSeconds,
		MaxFilePreviewSizeBytes: DefaultMaxFilePreviewSizeBytes,
	}
}
