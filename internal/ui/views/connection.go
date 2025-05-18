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
			components.NewConnectionDialog(ctx, "New connection", "", "", "", "", "", "", false, false, connection.AWSConnectionType,
				func(name, accessKey, secretKey, server, bucket, region string, useTLS bool, connectionType connection.ConnectionType) error {
					var newConn *connection.Connection
					switch connectionType {
					case connection.AWSConnectionType:
						newConn = connection.NewConnection(name, accessKey, secretKey, bucket, connection.AsAWSConnection(region))
					case connection.S3LikeConnectionType:
						newConn = connection.NewConnection(name, accessKey, secretKey, bucket, connection.AsS3LikeConnection(server, useTLS))
					}
					return ctx.ConnectionViewModel().SaveConnection(newConn)
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
