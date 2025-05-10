package views

import (
	"fmt"
	"strconv"

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
	themeSelector := widget.NewSelect([]string{"light", "dark", "system"}, func(s string) {
		ct, err := settings.NewColorThemeFromString(s)
		if err != nil {
			panic(err)
		}
		err = ctx.SettingsViewModel().ChangeColorTheme(ct)
		if err != nil {
			dialog.ShowError(err, ctx.Window())
		}
	})
	maxFilePreviewSizeEntry := widget.NewEntryWithData(
		binding.IntToString(ctx.SettingsViewModel().MaxFilePreviewSizeBytes()))
	currentTheme := ctx.SettingsViewModel().CurrentColorTheme()
	themeSelector.PlaceHolder = currentTheme.String()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Timeout (seconde)", Widget: timeoutEntry},
			{Text: "Color theme", Widget: themeSelector},
			{Text: "File size limit for preview (MB)", Widget: maxFilePreviewSizeEntry},
		},
		OnSubmit: func() {
			timeout, err := strconv.Atoi(timeoutEntry.Text)
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			if err := timeoutValidator(timeout); err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			maxFilePreviewSize, err := strconv.Atoi(maxFilePreviewSizeEntry.Text)
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			if err := maxFilePreviewSizeBytesValidator(maxFilePreviewSize); err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			s, err := settings.NewSettings(timeout, maxFilePreviewSize)
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
		SubmitText: "Save",
	}

	goToExplorerBtn := widget.NewButtonWithIcon(
		"View files",
		theme.NavigateBackIcon(),
		func() {
			ctx.Navigate(navigation.ExplorerRoute)
		},
	)

	return container.NewVBox(
		container.NewHBox(goToExplorerBtn),
		form,
	), nil
}

func maxFilePreviewSizeBytesValidator(maxFilePreviewSizeBytes int) error {
	if maxFilePreviewSizeBytes < 0 {
		return fmt.Errorf("max file preview size bytes must be greater than 0")
	}
	if maxFilePreviewSizeBytes > 1024*1024*1024*10 {
		return fmt.Errorf("max file preview size bytes must be less than 10GB")
	}
	return nil
}

func timeoutValidator(timeout int) error {
	if timeout < 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	if timeout > 1000 {
		return fmt.Errorf("timeout must be less than 1000 seconds")
	}
	return nil
}
