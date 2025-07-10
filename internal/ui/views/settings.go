package views

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/settings"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func GetSettingsView(ctx appcontext.AppContext) (*fyne.Container, error) {
	timeoutEntry := widget.NewEntryWithData(
		binding.IntToString(ctx.SettingsViewModel().TimeoutInSeconds()))
	themeSelector := widget.NewSelect(settings.AllColorThemesStr, func(s string) {
		if err := ctx.SettingsViewModel().ColorTheme().Set(s); err != nil {
			dialog.ShowError(err, ctx.Window())
		}
	})
	currentTheme, _ := ctx.SettingsViewModel().ColorTheme().Get()
	themeSelector.PlaceHolder = currentTheme

	maxFilePreviewSizeEntry := widget.NewEntryWithData(
		binding.IntToString(ctx.SettingsViewModel().MaxFilePreviewSizeMegaBytes()))

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Timeout (seconde)", Widget: timeoutEntry},
			{Text: "Color theme", Widget: themeSelector},
			{Text: "AttachedFile size limit for preview (MB)", Widget: maxFilePreviewSizeEntry},
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

	goToExplorerBtn := widget.NewButtonWithIcon(
		"View files",
		theme.NavigateBackIcon(),
		func() {
			ctx.Navigate(navigation.ExplorerRoute)
		},
	)

	exportConnectionsBtn := widget.NewButtonWithIcon(
		"Export connections as JSON",
		theme.DocumentSaveIcon(),
		func() {
			export, err := ctx.ConnectionViewModel().ExportAsJSON()
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}
			saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, ctx.Window())
					return
				}
				if writer == nil {
					return
				}
				defer writer.Close()
				_, writeErr := writer.Write(export.JSONData)
				if writeErr != nil {
					dialog.ShowError(writeErr, ctx.Window())
					return
				}
				msg := fmt.Sprintf("%d connection(s) exported as JSON", export.Count)
				dialog.ShowInformation("Export", msg, ctx.Window())
			}, ctx.Window())
			saveDialog.SetFileName("connections.json")
			saveDialog.Show()
		},
	)
	exportConnectionsBtn.Resize(fyne.NewSize(100, 100))

	return container.NewVBox(
		container.NewHBox(goToExplorerBtn),
		container.NewVBox(form, container.NewCenter(exportConnectionsBtn)),
	), nil
}
