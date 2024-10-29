package components

import (
	"fmt"
	"go2s3/internal/explorer"
	appcontext "go2s3/internal/ui/app/context"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	previewTabWidth = 2 // TODO: a ajouter dans les settings
)

func isStringPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) && r != '\n' && r != '\t' && r != '\r' {
			return false
		}
	}
	return true
}

func ShowFilePreviewDialog(ctx appcontext.AppContext, file *explorer.RemoteFile) {
	previewData := binding.NewString()
	loading := binding.NewBool()
	loading.Set(false)
	nbLines := binding.NewInt()

	go func() {
		loading.Set(true)
		defer loading.Set(false)
		fileContent, err := ctx.Vm().PreviewFile(file)
		if err != nil {
			dialog.ShowError(err, ctx.W())
			return
		}
		if !isStringPrintable(fileContent) {
			fileContent = "Binary file, no preview available."
		}
		if err = nbLines.Set(strings.Count(fileContent, "\n") + 1); err != nil {
			dialog.ShowError(fmt.Errorf("impossible to display line number: %s", err), ctx.W())
			return
		}
		if err = previewData.Set(fileContent); err != nil {
			dialog.ShowError(fmt.Errorf("file preview impossible: %s", err), ctx.W())
			return
		}
	}()

	preview := widget.NewEntryWithData(previewData)
	preview.OnChanged = func(_ string) {
		originalText, _ := previewData.Get()
		previewData.Set(originalText)
	}

	loadingBar := widget.NewProgressBarInfinite()
	loadingBar.Hide()

	loading.AddListener(binding.NewDataListener(func() {
		isLoading, _ := loading.Get()
		if isLoading {
			loadingBar.Show()
			preview.Hide()
		} else {
			loadingBar.Hide()
			preview.Show()
		}
	}))

	copyContentBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		fileContent, _ := previewData.Get()
		ctx.W().Clipboard().SetContent(fileContent)
	})

	nbLinesLabel := widget.NewLabel("")
	nbLines.AddListener(binding.NewDataListener(func() {
		nbLinesValue, _ := nbLines.Get()
		nbLinesLabel.SetText(fmt.Sprintf("%d lines", nbLinesValue))
	}))

	container := container.NewBorder(
		container.NewBorder(
			nil, nil,
			container.NewHBox(copyContentBtn),
			nbLinesLabel,
		),
		loadingBar,
		nil,
		nil,
		preview,
	)
	dial := dialog.NewCustom(
		file.Name(),
		"Close",
		container,
		ctx.W(),
	)
	dial.Resize(fyne.NewSize(700, 500))
	dial.Show()
}
