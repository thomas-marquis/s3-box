package csveditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Widget struct {
	widget.BaseWidget

	editor *Editor

	SaveBtn *widget.ToolbarAction
}

func newWidget(e *Editor) *Widget {
	w := &Widget{
		editor: e,
	}
	w.ExtendBaseWidget(w)

	e.ConfirmClose = func(onConfirm func(confirmed bool)) {
		dialog.ShowConfirm("Confirm close", "Are you sure you want to close the editor?", func(ok bool) {
			onConfirm(ok)
		}, e.Window())
	}

	e.Err.AddListener(binding.NewDataListener(func() {
		err, _ := e.Err.Get()
		if err == nil {
			return
		}
		dialog.ShowError(err, e.Window())
		e.Err.Set(nil) //nolint:errcheck
	}))

	return w
}

func (w *Widget) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	var cancelBtn *widget.Button
	var prevBtn, nextBtn *widget.Button

	table := widget.NewTable(
		func() (int, int) {
			nbLines := w.editor.Records.Length()
			if nbLines == 0 {
				return 0, 0
			}

			firstLine, err := w.editor.Records.GetValue(0)
			if err != nil {
				return nbLines, 0
			}

			nbCols := len(firstLine)

			return nbLines, nbCols
		},
		func() fyne.CanvasObject {
			cell := newCellEntry(w.editor.Records)
			cell.OnSave = w.editor.Save
			cell.OnClose = w.editor.RequestClose
			return cell
		},
		func(id widget.TableCellID, object fyne.CanvasObject) {
			cell := object.(*CellEntry)
			cell.UpdateCoords(id.Row, id.Col)

			rawVal, _ := w.editor.Records.GetValue(id.Row)
			cellVal := rawVal[id.Col]
			cell.SetText(cellVal)
		})

	table.HideSeparators = true
	table.Hide()

	w.editor.Records.AddListener(binding.NewDataListener(table.Refresh))

	w.editor.AddListener(listenerColumnsWidthKey, func() {
		cols, _ := w.editor.Columns.Get()
		for i, col := range cols {
			table.SetColumnWidth(i, float32(col))
		}
	})

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

	if isLoading, _ := w.editor.IsLoading.Get(); isLoading {
		loader.Start()
		loaderContainer.Show()
	} else {
		table.Show()
	}

	w.editor.IsLoading.AddListener(binding.NewDataListener(func() {
		isLoading, _ := w.editor.IsLoading.Get()
		if isLoading {
			loaderContainer.Show()
			loader.Start()
			table.Hide()
		} else {
			loaderContainer.Hide()
			loader.Stop()
			table.Show()
			table.Refresh()
			if w.editor.Paginator.HasNext() {
				nextBtn.Enable()
			}
		}
	}))

	w.SaveBtn = widget.NewToolbarAction(theme.DocumentSaveIcon(), w.editor.Save)

	pageLabel := widget.NewLabelWithData(w.editor.PageLabel)
	pageLabel.Alignment = fyne.TextAlignCenter

	prevBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		w.editor.PrevPage()
		if w.editor.Paginator.CurrentIndex == 0 {
			prevBtn.Disable()
		}
		if w.editor.Paginator.HasNext() {
			nextBtn.Enable()
		}
	})
	prevBtn.Disable()

	nextBtn = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if !w.editor.NextPage() {
			nextBtn.Disable()
		}
		if w.editor.Paginator.CurrentIndex > 0 {
			prevBtn.Enable()
		}
	})
	nextBtn.Disable()

	pagination := container.NewHBox(prevBtn, pageLabel, nextBtn)

	top := container.NewBorder(nil, nil,
		container.NewHBox(widget.NewToolbar(w.SaveBtn), pagination),
		widget.NewLabelWithData(w.editor.StatusLabel),
	)

	bottom := container.NewBorder(nil, nil,
		nil, nil,
		loaderContainer,
	)

	c := container.NewBorder(top, bottom,
		nil, nil,
		table)

	return widget.NewSimpleRenderer(c)
}
