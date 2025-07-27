package components

import (
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"strings"
	"unicode"

	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

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

func ShowFilePreviewDialog(ctx appcontext.AppContext, file *directory.File) {
	previewData := binding.NewString()
	loading := binding.NewBool()
	loading.Set(false)
	nbLines := binding.NewInt()

	go func() {
		fyne.Do(func() {
			loading.Set(true)
			defer loading.Set(false)

			fileContent, err := ctx.ExplorerViewModel().GetFileContent(file)
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}

			f, err := fileContent.Open()
			if err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}
			defer f.Close()

			var contentBytes []byte
			if _, err := f.Read(contentBytes); err != nil {
				dialog.ShowError(err, ctx.Window())
				return
			}
			contentStr := string(contentBytes)

			if !isStringPrintable(contentStr) {
				contentStr = "Binary file, no preview available."
			}
			if err = nbLines.Set(strings.Count(contentStr, "\n") + 1); err != nil {
				dialog.ShowError(fmt.Errorf("impossible to display line number: %s", err), ctx.Window())
				return
			}
			if err = previewData.Set(contentStr); err != nil {
				dialog.ShowError(fmt.Errorf("file preview impossible: %s", err), ctx.Window())
				return
			}
		})
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
		ctx.Window().Clipboard().SetContent(fileContent)
	})

	nbLinesLabel := widget.NewLabel("")
	nbLines.AddListener(binding.NewDataListener(func() {
		nbLinesValue, _ := nbLines.Get()
		nbLinesLabel.SetText(fmt.Sprintf("%d lines", nbLinesValue))
	}))

	dial := dialog.NewCustom(
		file.Name().String(),
		"Close",
		container.NewBorder(
			container.NewBorder(
				nil, nil,
				container.NewHBox(copyContentBtn),
				nbLinesLabel,
			),
			loadingBar,
			nil,
			nil,
			preview,
		),
		ctx.Window(),
	)
	dial.Resize(fyne.NewSize(700, 500))
	dial.Show()
}
