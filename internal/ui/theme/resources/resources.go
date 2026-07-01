package resources

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed logo-wbg.png
var resourceLogoWbgPngData []byte
var resourceLogoWbgPng = &fyne.StaticResource{
	StaticName:    "logo-wbg.png",
	StaticContent: resourceLogoWbgPngData,
}

func NewAppLogo() fyne.Resource {
	return resourceLogoWbgPng
}
