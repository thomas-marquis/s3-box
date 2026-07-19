package csveditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	cellPadding = 50
)

type Widget struct {
	widget.BaseWidget

	editor *csvEditor

	SaveBtn *widget.ToolbarAction
}

func newWidget(editor *csvEditor) *Widget {
	w := &Widget{
		editor: editor,
	}

	editor.ConfirmClose = func(onConfirm func(confirmed bool)) {
		dialog.ShowConfirm("Confirm close", "Are you sure you want to close the editor?", func(ok bool) {
			onConfirm(ok)
		}, editor.Window())
	}

	w.ExtendBaseWidget(w)
	return w
}

func (w *Widget) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	var table *widget.Table
	table = widget.NewTable(
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
			cell.OnClose = w.editor.Close
			return cell
		},
		func(id widget.TableCellID, object fyne.CanvasObject) {
			cell := object.(*CellEntry)
			cell.UpdateCoords(id.Row, id.Col)

			rawVal, _ := w.editor.Records.GetValue(id.Row)
			cellVal := rawVal[id.Col]
			cell.SetText(cellVal)

			th := w.Theme()
			cellSize := fyne.MeasureText(cellVal, th.Size(theme.SizeNameText), fyne.TextStyle{})
			cell.Resize(fyne.NewSize(cellSize.Width, cell.Size().Height))
		})

	table.HideSeparators = true
	table.Hide()

	w.editor.Records.AddListener(binding.NewDataListener(table.Refresh))

	w.editor.Columns.AddListener(binding.NewDataListener(func() {
		cols, _ := w.editor.Columns.Get()
		for i, col := range cols {
			table.SetColumnWidth(i, col.Width)
		}
	}))

	loader := widget.NewProgressBarInfinite()

	var cancelBtn *widget.Button
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
		} else {
			loaderContainer.Hide()
			loader.Stop()
			table.Show()
		}
	}))

	w.SaveBtn = widget.NewToolbarAction(theme.DocumentSaveIcon(), w.editor.Save)
	toolbar := widget.NewToolbar(w.SaveBtn)

	top := container.NewBorder(nil, nil,
		toolbar,
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
