package theme

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
)

// GetDefaultByVariant returns the default theme for the given variant (light or dark.
func GetDefaultByVariant(variant fyne.ThemeVariant) fyne.Theme {
	switch variant {
	case theme.VariantDark:
		return &appThemeDark{}
	case theme.VariantLight:
		return &appThemeLight{}
	default:
		if fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark {
			return &appThemeDark{}
		}
		return &appThemeLight{}
	}
}

// GetByName returns the theme for the given name.
func GetByName(themeName settings.ColorTheme) fyne.Theme {
	switch themeName {
	case settings.ColorThemeLight:
		return GetDefaultByVariant(theme.VariantLight)
	case settings.ColorThemeDark:
		return GetDefaultByVariant(theme.VariantDark)
	case settings.ColorThemeSystem:
		return theme.DefaultTheme()
	}

	return theme.DefaultTheme()
}

type appThemeLight struct{}

var _ fyne.Theme = (*appThemeLight)(nil)

func (t appThemeLight) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return hexToNRGBA("#FFF8F2")
	case "onBackground":
		return hexToNRGBA("#1F1B13")
	case theme.ColorNameButton:
		return hexToNRGBA("#E8C26C")
	case theme.ColorNameDisabledButton:
		return hexToNRGBA("#D0C5B4")
	case theme.ColorNameDisabled:
		return hexToNRGBA("#FFF8F2")
	case theme.ColorNameError:
		return hexToNRGBA("#BA1A1A")
	case theme.ColorNameFocus:
		return hexToNRGBA("#775A0B80")
	case theme.ColorNameForeground:
		return hexToNRGBA("#1F1B13")
	case theme.ColorNameForegroundOnError:
		return hexToNRGBA("#FFFFFF")
	case theme.ColorNameForegroundOnPrimary:
		return hexToNRGBA("#FFFFFF")
	case theme.ColorNameForegroundOnSuccess:
		return hexToNRGBA("#FFFFFF")
	case theme.ColorNameForegroundOnWarning:
		return hexToNRGBA("#1F1B13")
	case theme.ColorNameHeaderBackground:
		return hexToNRGBA("#F1E7D9")
	case theme.ColorNameHover:
		return hexToNRGBA("#00000040")
	case theme.ColorNameHyperlink:
		return hexToNRGBA("#775A0B")
	case theme.ColorNameInputBackground:
		return hexToNRGBA("#FCF2E5")
	case theme.ColorNameInputBorder:
		return hexToNRGBA("#7F7667")
	case theme.ColorNameMenuBackground:
		return hexToNRGBA("#F6EDDF")
	case theme.ColorNameOverlayBackground:
		return hexToNRGBA("#EBE1D4EE")
	case theme.ColorNamePlaceHolder:
		return hexToNRGBA("#4D4639")
	case theme.ColorNamePressed:
		return hexToNRGBA("#775A0B33")
	case theme.ColorNamePrimary:
		return hexToNRGBA("#775A0B")
	case theme.ColorNameScrollBar:
		return hexToNRGBA("#7F7667")
	case theme.ColorNameScrollBarBackground:
		return hexToNRGBA("#F6EDDF")
	case theme.ColorNameSelection:
		return hexToNRGBA("#FFDF9C")
	case theme.ColorNameSeparator:
		return hexToNRGBA("#D0C5B4")
	case theme.ColorNameShadow:
		return hexToNRGBA("#00000033")
	case theme.ColorNameSuccess:
		return hexToNRGBA("#2E7D32")
	case theme.ColorNameWarning:
		return hexToNRGBA("#FF9800")
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (t appThemeLight) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t appThemeLight) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t appThemeLight) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

type appThemeDark struct{}

var _ fyne.Theme = (*appThemeDark)(nil)

func (t appThemeDark) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return hexToNRGBA("#17130B")
	case "onBackground":
		return hexToNRGBA("#EBE1D4")
	case theme.ColorNameButton:
		return hexToNRGBA("#775A0B")
	case theme.ColorNamePrimary:
		return hexToNRGBA("#E8C26C")
	case theme.ColorNameDisabledButton:
		return hexToNRGBA("#4D4639")
	case theme.ColorNameDisabled:
		return hexToNRGBA("#FFF8F2")
	case theme.ColorNameError:
		return hexToNRGBA("#FFB4AB")
	case theme.ColorNameFocus:
		return hexToNRGBA("#E8C26C80")
	case theme.ColorNameForeground:
		return hexToNRGBA("#EBE1D4")
	case theme.ColorNameForegroundOnError:
		return hexToNRGBA("#690005")
	case theme.ColorNameForegroundOnPrimary:
		return hexToNRGBA("#FFFFFF")
	case theme.ColorNameForegroundOnSuccess:
		return hexToNRGBA("#212121")
	case theme.ColorNameForegroundOnWarning:
		return hexToNRGBA("#212121")
	case theme.ColorNameHeaderBackground:
		return hexToNRGBA("#2E2921")
	case theme.ColorNameHover:
		return hexToNRGBA("#EBE1D426")
	case theme.ColorNameHyperlink:
		return hexToNRGBA("#E8C26C")
	case theme.ColorNameInputBackground:
		return hexToNRGBA("#1F1B13")
	case theme.ColorNameInputBorder:
		return hexToNRGBA("#999080")
	case theme.ColorNameMenuBackground:
		return hexToNRGBA("#231F17")
	case theme.ColorNameOverlayBackground:
		return hexToNRGBA("#39342BEE")
	case theme.ColorNamePlaceHolder:
		return hexToNRGBA("#D0C5B4")
	case theme.ColorNamePressed:
		return hexToNRGBA("#E8C26C33")
	case theme.ColorNameScrollBar:
		return hexToNRGBA("#999080")
	case theme.ColorNameScrollBarBackground:
		return hexToNRGBA("#231F17")
	case theme.ColorNameSelection:
		return hexToNRGBA("#5B4300")
	case theme.ColorNameSeparator:
		return hexToNRGBA("#4D4639")
	case theme.ColorNameShadow:
		return hexToNRGBA("#00000066")
	case theme.ColorNameSuccess:
		return hexToNRGBA("#81C784")
	case theme.ColorNameWarning:
		return hexToNRGBA("#FFB74D")
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (t appThemeDark) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t appThemeDark) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t appThemeDark) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// hexToNRGBA convert an hexadecimal color code (ex: "#RRGGBB" ou "#RRGGBBAA") into a color.NRGBA.
func hexToNRGBA(hex string) color.NRGBA {
	hex = strings.TrimPrefix(hex, "#")

	var r, g, b, a uint64
	var err error

	switch len(hex) {
	case 6: // RRGGBB
		if r, err = strconv.ParseUint(hex[0:2], 16, 8); err != nil {
			return color.NRGBA{}
		}
		if g, err = strconv.ParseUint(hex[2:4], 16, 8); err != nil {
			return color.NRGBA{}
		}
		if b, err = strconv.ParseUint(hex[4:6], 16, 8); err != nil {
			return color.NRGBA{}
		}
		a = 0xff
	case 8: // RRGGBBAA
		if r, err = strconv.ParseUint(hex[0:2], 16, 8); err != nil {
			return color.NRGBA{}
		}
		if g, err = strconv.ParseUint(hex[2:4], 16, 8); err != nil {
			return color.NRGBA{}
		}
		if b, err = strconv.ParseUint(hex[4:6], 16, 8); err != nil {
			return color.NRGBA{}
		}
		if a, err = strconv.ParseUint(hex[6:8], 16, 8); err != nil {
			return color.NRGBA{}
		}
	default:
		return color.NRGBA{}
	}

	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
