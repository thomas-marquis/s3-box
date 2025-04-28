package views

import (
	"strconv"

	"github.com/thomas-marquis/s3-box/internal/settings"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func GetSettingsView(ctx appcontext.AppContext) (*fyne.Container, error) {
	timeoutEntry := widget.NewEntry()
	timeoutEntry.SetText("30") // Default value

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Timeout (seconde)", Widget: timeoutEntry},
		},
		OnSubmit: func() {
			timeout, err := strconv.Atoi(timeoutEntry.Text)
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			s, err := settings.NewSettings(timeout)
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			if err := ctx.SettingsViewModel().Save(s); err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			dialog.ShowInformation("Done", "Settings Saved", ctx.Window())
		},
	}

	backButton := widget.NewButton("Retour", func() {
		ctx.Navigate(navigation.ExplorerRoute)
	})

	return container.NewVBox(
		form,
		backButton,
	), nil
}

