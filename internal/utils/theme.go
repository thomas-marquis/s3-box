package utils

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	apptheme "github.com/thomas-marquis/s3-box/internal/ui/theme"
)

func MapFyneColorTheme(colorTheme settings.ColorTheme) fyne.Theme {
	switch colorTheme {
	case settings.ColorThemeLight:
		return apptheme.Get(theme.VariantLight)
	case settings.ColorThemeDark:
		return apptheme.Get(theme.VariantDark)
	case settings.ColorThemeSystem:
		return theme.DefaultTheme()
	}

	return theme.DefaultTheme()
}
