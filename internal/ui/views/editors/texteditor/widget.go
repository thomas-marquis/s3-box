package texteditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TextEditor struct {
	widget.BaseWidget

	editor *textEditor

	TextEntry *textContentEntry
	SaveBtn   *widget.ToolbarAction
}

func newWidget(e *textEditor) fyne.CanvasObject {
	w := &TextEditor{
		editor: e,
	}
	w.ExtendBaseWidget(w)

	e.Err.AddListener(binding.NewDataListener(func() {
		err, _ := e.Err.Get()
		if err == nil {
			return
		}
		dialog.ShowError(err, e.Window())
		e.Err.Set(nil) //nolint:errcheck
	}))

	e.ConfirmClose = func(onConfirm func(confirmed bool)) {
		dialog.ShowConfirm("Confirm close", "Are you sure you want to close the editor?", func(ok bool) {
			onConfirm(ok)
		}, e.Window())
	}

	return w
}

func (w *TextEditor) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	textEntry := newTextEditorEntry(w.editor.Save, w.editor.RequestClose)
	w.TextEntry = textEntry
	textEntry.Bind(w.editor.ContentStr)

	var cancelBtn *widget.Button
	w.SaveBtn = widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
		cancelBtn.Enable()
		w.editor.Save(textEntry.Text)
	})
	toolbar := widget.NewToolbar(w.SaveBtn)

	loader := widget.NewProgressBarInfinite()
	cancelBtn = widget.NewButton("Cancel", func() {
		cancelBtn.Disable()
		w.editor.StatusLabel.Set("cancelling...") //nolint:errcheck
		w.editor.Cancel()
	})
	loaderContainer := container.NewBorder(
		nil, nil, nil,
		cancelBtn, loader,
	)
	loader.Stop()
	loaderContainer.Hide()

	w.editor.IsLoading.AddListener(binding.NewDataListener(func() {
		isLoading, _ := w.editor.IsLoading.Get()
		if isLoading {
			loaderContainer.Show()
			loader.Start()
		} else {
			loaderContainer.Hide()
			loader.Stop()
		}
	}))

	bottomBar := container.NewBorder(nil, nil,
		widget.NewButtonWithIcon("Save & Exit", theme.DocumentSaveIcon(), func() {
			w.editor.SaveThenExit(textEntry.Text)
		}), nil,
		loaderContainer,
	)

	c := container.NewBorder(
		container.NewBorder(nil, nil,
			toolbar,
			widget.NewLabelWithData(w.editor.StatusLabel)),
		bottomBar,
		nil, nil,
		textEntry)

	return widget.NewSimpleRenderer(c)
}
