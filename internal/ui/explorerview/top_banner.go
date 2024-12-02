package explorerview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
)

func makeNoConnectionTopBanner(ctx appcontext.AppContext) *fyne.Container {
	return container.NewVBox(
		container.NewCenter(widget.NewLabel("No connection selected, please select a connection in the settings menu")),
		container.NewCenter(widget.NewButton("Manage connections", func() {
			ctx.Navigate(navigation.ConnectionRoute)
		})),
	)
}
