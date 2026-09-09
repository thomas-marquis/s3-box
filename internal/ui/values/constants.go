package values

// theme

const (
	ColorThemeLight  = "light"
	ColorThemeDark   = "dark"
	ColorThemeSystem = "system"
)

var (
	AllColorThemesStr = []string{ColorThemeLight, ColorThemeDark, ColorThemeSystem}
)

// settings

const (
	SettingColorTheme            = "app.colorTheme"
	SettingEditFileSizeLimitByte = "app.editFileSizeLimitByte"
	SettingTimeoutSec            = "app.timeoutSec"
	SettingFileSelectors         = "settings.editor.selectors"
)

// hard-coded config

const (
	ExplorerNbWorkers = 3
)
