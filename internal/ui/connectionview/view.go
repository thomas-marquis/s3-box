package connectionview

import (
	"github.com/thomas-marquis/s3-box/internal/connection"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func GetView(ctx appcontext.AppContext) (*fyne.Container, error) {
	connLine := newConnectionLine()
	connectionsList := widget.NewListWithData(
		ctx.Vm().Connections(),
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
			newConnectionDialog(ctx, "New connection", "", "", "", "", "", "", false, false,
				func(name, accessKey, secretKey, server, bucket, region string, useTLS bool) error {
					conn := connection.NewConnection(name, server, accessKey, secretKey, bucket, useTLS, region)
					return ctx.Vm().SaveConnection(conn)
				}).Show()
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
