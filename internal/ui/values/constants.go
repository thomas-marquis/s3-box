package values

const (
	ColorThemeLight  = "light"
	ColorThemeDark   = "dark"
	ColorThemeSystem = "system"
)

var (
	AllColorThemesStr = []string{ColorThemeLight, ColorThemeDark, ColorThemeSystem}
)

const (
	SettingColorTheme            = "app.colorTheme"
	SettingEditFileSizeLimitByte = "app.editFileSizeLimitByte"
	SettingTimeoutSec            = "app.timeoutSec"
)
