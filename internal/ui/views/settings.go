package views

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	fyne_widget "fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

// GetSettingsView creates and returns a settings view container with form elements and buttons for user interaction.
// It implements the navigation.View signature type.
// Returns the constructed fyne.Container and an error if a problem occurs during the process.
func GetSettingsView(ctx appcontext.AppContext) (*fyne.Container, error) {
	timeoutEntry := fyne_widget.NewEntryWithData(
		binding.IntToString(ctx.SettingsViewModel().TimeoutInSeconds()))

	themeSelector := fyne_widget.NewSelect(settings.AllColorThemesStr, func(s string) {
		if err := ctx.SettingsViewModel().ColorTheme().Set(s); err != nil {
			dialog.ShowError(err, ctx.Window())
		}
	})
	currentTheme, _ := ctx.SettingsViewModel().ColorTheme().Get()
	themeSelector.PlaceHolder = currentTheme

	sizeEntry := fyne_widget.NewEntryWithData(
		binding.IntToString(ctx.SettingsViewModel().FileSizeLimitKB()))

	form := &fyne_widget.Form{
		Items: []*fyne_widget.FormItem{
			{Text: "Color theme", Widget: themeSelector},
			{Text: "Preview/edit file size limit (KB)", Widget: sizeEntry},
			{Text: "Timeout (seconds)", Widget: timeoutEntry},
		},
		OnSubmit: func() {
			if err := ctx.SettingsViewModel().Save(); err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			dialog.ShowInformation("Done", "Settings Saved", ctx.Window())
		},
		SubmitText: "Save",
	}

	exportConnectionsBtn := fyne_widget.NewButtonWithIcon(
		"Export connections as JSON",
		theme.DocumentSaveIcon(),
		func() {
			saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, ctx.Window())
					return
				}
				if writer == nil {
					return
				}
				defer writer.Close() //nolint:errcheck

				if err := ctx.ConnectionViewModel().ExportAsJSON(writer); err != nil {
					dialog.ShowError(err, ctx.Window())
					return
				}

				deck := ctx.ConnectionViewModel().Deck()
				msg := fmt.Sprintf("%d connection(s) exported as JSON", len(deck.Get()))
				dialog.ShowInformation("Export", msg, ctx.Window())
			}, ctx.Window())
			saveDialog.SetFileName("connections.json")
			saveDialog.Show()
		},
	)
	exportConnectionsBtn.Resize(fyne.NewSize(100, 100))

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeading("Settings"),
			fyne_widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(
			container.NewGridWrap(fyne.NewSize(700, 400), container.NewVBox(
				form,
				container.NewHBox(exportConnectionsBtn),
			)),
		),
	), nil
}
