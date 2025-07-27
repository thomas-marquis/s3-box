package views

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func GetConnectionView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	deck := appCtx.ConnectionViewModel().Deck()
	connLine := components.NewConnectionLine(deck)
	connectionsList := widget.NewListWithData(
		appCtx.ConnectionViewModel().Connections(),
		func() fyne.CanvasObject {
			return connLine.Raw()
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			i, _ := di.(binding.Untyped).Get()
			conn, _ := i.(*connection_deck.Connection)
			connLine.Update(appCtx, obj, conn)
		},
	)
	createBtn := widget.NewButtonWithIcon(
		"New connection",
		theme.ContentAddIcon(),
		func() {
			components.NewConnectionDialog(appCtx,
				"New connection",
				connection_deck.Connection{},
				false,
				appCtx.ConnectionViewModel().Create).Show()
		})

	goToExplorerBtn := widget.NewButtonWithIcon(
		"View files",
		theme.NavigateBackIcon(),
		func() {
			appCtx.Navigate(navigation.ExplorerRoute)
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
