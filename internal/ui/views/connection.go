package views

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/u"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
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
		u.Skip(vm.ErrorMessage().Set(""))
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

	exportBtn := fyne_widget.NewButtonWithIcon(
		"Export",
		theme.DocumentSaveIcon(),
		func() {
			saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, appCtx.Window())
					return
				}
				if writer == nil {
					return
				}
				defer u.SkipD(writer.Close)

				if err := vm.ExportAsJSON(writer); err != nil {
					dialog.ShowError(err, appCtx.Window())
					return
				}

				deck := appCtx.State().Connection().Deck()
				msg := fmt.Sprintf("%d connection(s) exported as JSON", len(deck.Get()))
				dialog.ShowInformation("Export", msg, appCtx.Window())
			}, appCtx.Window())
			saveDialog.SetFileName("connections.json")
			saveDialog.Show()
		},
	)

	importBtn := fyne_widget.NewButtonWithIcon("Import", theme.UploadIcon(),
		func() {
			d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, appCtx.Window())
					return
				}
				if reader == nil {
					return
				}

				if err := vm.ImportFromJSON(reader); err != nil {
					dialog.ShowError(err, appCtx.Window())
					return
				}
			}, appCtx.Window())
			d.SetTitleText("Select a valid JSON file to import")
			d.Show()
		})

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeading("Manage connections"),
			fyne_widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(
			container.NewBorder(
				nil,
				container.NewBorder(nil, nil,
					nil, container.NewHBox(exportBtn, importBtn),
					container.NewCenter(createBtn)),
				nil,
				nil,
				connectionsList,
			),
		),
	), nil
}
