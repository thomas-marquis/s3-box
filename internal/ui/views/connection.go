package views

import (
	"github.com/thomas-marquis/s3-box/internal/connection"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func GetConnectionView(ctx appcontext.AppContext) (*fyne.Container, error) {
	connLine := components.NewConnectionLine()
	connectionsList := widget.NewListWithData(
		ctx.ConnectionViewModel().Connections(),
		func() fyne.CanvasObject {
			return connLine.Raw()
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			i, _ := di.(binding.Untyped).Get()
			conn, _ := i.(*connection.Connection)
			connLine.Update(ctx, obj, conn)
		},
	)
	createBtn := widget.NewButtonWithIcon(
		"New connection",
		theme.ContentAddIcon(),
		func() {
			components.NewConnectionDialog(ctx,
				"New connection",
				*connection.NewEmptyConnection(),
				false,
				ctx.ConnectionViewModel().Save).Show()
		})

	goToExplorerBtn := widget.NewButtonWithIcon(
		"View files",
		theme.NavigateBackIcon(),
		func() {
			ctx.Navigate(navigation.ExplorerRoute)
		},
	)

	mainContainer := container.NewBorder(
		container.NewHBox(goToExplorerBtn),
		container.NewCenter(createBtn),
		nil,
		nil,
		connectionsList,
	)
	return mainContainer, nil
}
