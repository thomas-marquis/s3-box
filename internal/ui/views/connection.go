package views

import (
	"errors"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
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
	vm := appCtx.ConnectionViewModel()

	vm.ErrorMessage().AddListener(binding.NewDataListener(func() {
		msg, _ := vm.ErrorMessage().Get()
		if msg == "" {
			return
		}
		dialog.ShowError(errors.New(msg), appCtx.Window())
		vm.ErrorMessage().Set("") //nolint:errcheck
	}))

	createBtn := fyne_widget.NewButtonWithIcon(
		"New connection",
		theme.ContentAddIcon(),
		widget.NewConnectionForm(appCtx,
			&connection_deck.Connection{},
			false,
			func(name, accessKey, secretKey, bucket string,
				options ...connection_deck.ConnectionOption) {
				vm.Create(name, accessKey, secretKey, bucket, options...)
			},
		).AsDialog("New connection").Show)

	goToExplorerBtn := fyne_widget.NewButtonWithIcon(
		"View files",
		theme.NavigateBackIcon(),
		func() {
			if _, err := appCtx.Navigate(navigation.ExplorerRoute); err != nil { //nolint:staticcheck
				// TODO: handle error
			}
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
