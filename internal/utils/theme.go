package utils

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

func MapFyneColorTheme(colorTheme settings.ColorTheme) fyne.Theme {
	switch colorTheme {
	case settings.ColorThemeLight:
		return theme.LightTheme()
	case settings.ColorThemeDark:
		return theme.DarkTheme()
	case settings.ColorThemeSystem:
		return theme.DefaultTheme()
	}

	return theme.DefaultTheme()
}
