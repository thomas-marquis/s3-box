package views

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fyne_widget "fyne.io/fyne/v2/widget"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/values"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

// GetSettingsView creates and returns a settings view container with form elements and buttons for user interaction.
// It implements the navigation.View signature type.
// Returns the constructed fyne.Container and an error if a problem occurs during the process.
func GetSettingsView(ctx appcontext.AppContext) (*fyne.Container, error) {
	timeoutEntry := widget.NewNumericalEntry[time.Duration](time.Second)
	timeoutEntry.Bind(ctx.State().Settings().Timeout())

	themeSelector := fyne_widget.NewSelectWithData(
		values.AllColorThemesStr, ctx.State().Settings().ColorTheme())
	themeSelector.PlaceHolder = "Select theme"

	sizeEntry := widget.NewNumericalEntry[uint64](values.KiB)
	sizeEntry.Bind(ctx.State().Settings().EditorFileSizeLimitBytes())

	form := &fyne_widget.Form{
		Items: []*fyne_widget.FormItem{
			{Text: "Color theme", Widget: themeSelector},
			{Text: "Preview/edit file size limit (KB)", Widget: sizeEntry},
			{Text: "Timeout (seconds)", Widget: timeoutEntry},
		},
		SubmitText: "Save",
		OnSubmit:   ctx.SettingsViewModel().Save,
		CancelText: "Cancel",
		OnCancel: func() {
			if !ctx.State().Settings().Get().HasPendingEvents() {
				return
			}
			dialog.ShowConfirm("Cancel", "Are you sure you want to cancel all unsaved changes?", func(confirmed bool) {
				if confirmed {
					ctx.SettingsViewModel().Cancel()
				}
			}, ctx.Window())
		},
	}

	statusLabel := fyne_widget.NewLabelWithData(ctx.State().Settings().StatusMessage())

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeading("Settings"),
			fyne_widget.NewSeparator(),
		), nil, nil, nil,
		container.NewVBox(
			container.NewBorder(nil, nil, nil, statusLabel),
			container.NewPadded(
				container.NewGridWrap(fyne.NewSize(700, 400), form),
			),
		),
	), nil
}
