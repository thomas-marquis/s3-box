package settings

import "errors"

type ColorTheme int

const (
	ColorThemeLight ColorTheme = iota
	ColorThemeDark
	ColorThemeSystem
)

const (
	ColorThemeLightStr  = "light"
	ColorThemeDarkStr   = "dark"
	ColorThemeSystemStr = "system"
)

var AllColorThemesStr = []string{ColorThemeLightStr, ColorThemeDarkStr, ColorThemeSystemStr}

func (c ColorTheme) String() string {
	switch c {
	case ColorThemeLight:
		return ColorThemeLightStr
	case ColorThemeDark:
		return ColorThemeDarkStr
	case ColorThemeSystem:
		return ColorThemeSystemStr
	default:
		return "unknown"
	}
}

func NewColorThemeFromString(s string) (ColorTheme, error) {
	switch s {
	case ColorThemeLightStr:
		return ColorThemeLight, nil
	case ColorThemeDarkStr:
		return ColorThemeDark, nil
	case ColorThemeSystemStr:
		return ColorThemeSystem, nil
	default:
		return -1, errors.New("invalid color theme")
	}
}

type Settings struct {
	TimeoutInSeconds        int
	Color                   ColorTheme
	MaxFilePreviewSizeBytes int64
}

var ErrInvalidTimeout = errors.New("timeout must be positive")

const (
	DefaultTimeoutInSeconds        = 15
	DefaultMaxFilePreviewSizeBytes = 1024 * 1024 * 5 // 5MB
)

func NewSettings(timeoutInSeconds int, maxFilePreviewSizeBytes int64) (Settings, error) {
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
