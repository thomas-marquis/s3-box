package views

import (
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	fyne_widget "fyne.io/fyne/v2/widget"
)

func GetConnectionView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	connectionsList := widget.NewConnectionList(appCtx)

	createBtn := fyne_widget.NewButtonWithIcon(
		"New connection",
		theme.ContentAddIcon(),
		widget.NewConnectionForm(appCtx,
			&connection_deck.Connection{},
			false,
			appCtx.ConnectionViewModel().Create,
		).AsDialog("New connection").Show)

	goToExplorerBtn := fyne_widget.NewButtonWithIcon(
		"View files",
		theme.NavigateBackIcon(),
		func() {
			appCtx.Navigate(navigation.ExplorerRoute)
		},
	)

	return container.NewBorder(
		container.NewHBox(goToExplorerBtn),
		container.NewCenter(createBtn),
		nil,
		nil,
		connectionsList,
	), nil
}
