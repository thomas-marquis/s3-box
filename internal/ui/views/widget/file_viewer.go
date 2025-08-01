package widget

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"strings"
	"unicode"
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

type FileViewer struct {
	widget.BaseWidget

	appCtx appcontext.AppContext

	copyContentBtn *widget.Button
	nbLinesLabel   *widget.Label
	loadingBar     *widget.ProgressBarInfinite
	preview        *widget.Entry
}

func NewFileViewer(appCtx appcontext.AppContext, file *directory.File) *FileViewer {
	previewData := binding.NewString()
	loading := binding.NewBool()
	loading.Set(false)
	nbLines := binding.NewInt()

	go func() {
		fyne.Do(func() {
			loading.Set(true)
			defer loading.Set(false)

			fileContent, err := appCtx.ExplorerViewModel().GetFileContent(file)
			if err != nil {
				dialog.ShowError(err, appCtx.Window())
				return
			}

			f, err := fileContent.Open()
			if err != nil {
				dialog.ShowError(err, appCtx.Window())
				return
			}
			defer f.Close()

			contentBytes := make([]byte, file.SizeBytes())
			if _, err := f.Read(contentBytes); err != nil {
				dialog.ShowError(err, appCtx.Window())
				return
			}
			contentStr := string(contentBytes)

			if !isStringPrintable(contentStr) {
				contentStr = "Binary file, no preview available."
			}
			if err = nbLines.Set(strings.Count(contentStr, "\n") + 1); err != nil {
				dialog.ShowError(fmt.Errorf("impossible to display line number: %s", err), appCtx.Window())
				return
			}
			if err = previewData.Set(contentStr); err != nil {
				dialog.ShowError(fmt.Errorf("file preview impossible: %s", err), appCtx.Window())
				return
			}
		})
	}()

	copyContentBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		fileContent, _ := previewData.Get()
		appCtx.Window().Clipboard().SetContent(fileContent)
	})

	nbLinesLabel := widget.NewLabel("")
	nbLines.AddListener(binding.NewDataListener(func() {
		nbLinesValue, _ := nbLines.Get()
		nbLinesLabel.SetText(fmt.Sprintf("%d lines", nbLinesValue))
	}))

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

	w := &FileViewer{
		appCtx:         appCtx,
		copyContentBtn: copyContentBtn,
		nbLinesLabel:   nbLinesLabel,
		loadingBar:     loadingBar,
		preview:        preview,
	}

	w.ExtendBaseWidget(w)
	return w
}

func (w *FileViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		container.NewBorder(
			nil, nil,
			container.NewHBox(w.copyContentBtn),
			w.nbLinesLabel,
		),
		w.loadingBar,
		nil,
		nil,
		w.preview,
	))
}
