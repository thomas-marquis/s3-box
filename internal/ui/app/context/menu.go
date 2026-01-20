package appcontext

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
)

type Menu struct {
	Label       string
	IconFactory func() fyne.Resource
	Route       navigation.Route
	View        View
	Index       uint8
}
