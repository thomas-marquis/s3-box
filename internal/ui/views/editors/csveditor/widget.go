package csveditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

	w.ExtendBaseWidget(w)
	return w
}

func (w *Widget) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	var table *widget.Table
	table = widget.NewTable(
		func() (rows int, cols int) {
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
			cell := widget.NewEntry()
			cell.Scroll = fyne.ScrollNone
			cell.MultiLine = false
			return cell
		}, func(id widget.TableCellID, object fyne.CanvasObject) {
			cell := object.(*widget.Entry)
			rawVal, _ := w.editor.Records.GetValue(id.Row)
			cellVal := rawVal[id.Col]
			cell.SetText(cellVal)

			th := w.Theme()
			cellSize := fyne.MeasureText(cellVal, th.Size(theme.SizeNameText), fyne.TextStyle{})
			cell.Resize(fyne.NewSize(cellSize.Width, cell.Size().Height))
		})

	table.HideSeparators = true

	w.editor.Records.AddListener(binding.NewDataListener(table.Refresh))

	w.editor.Columns.AddListener(binding.NewDataListener(func() {
		cols, _ := w.editor.Columns.Get()
		for i, col := range cols {
			table.SetColumnWidth(i, col.Width+50)
		}
	}))

	loader := widget.NewProgressBarInfinite()
	//var cancelBtn *widget.Button // TODO
	loaderContainer := container.NewBorder(
		nil, nil, nil,
		nil, loader,
	)
	loader.Stop()
	loaderContainer.Hide()

	if isLoading, _ := w.editor.IsLoading.Get(); isLoading {
		loader.Start()
		loaderContainer.Show()
	}

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

	w.SaveBtn = widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
		w.editor.Save()
	})
	toolbar := widget.NewToolbar(w.SaveBtn)

	top := container.NewBorder(nil, nil,
		toolbar,
		nil,
	)

	bottom := container.NewBorder(nil, nil,
		//widget.NewButtonWithIcon("Save & Exit", theme.DocumentSaveIcon(), func() {
		//	w.editor.SaveThenExit(textEntry.Text)
		//}), nil,
		nil, nil,
		loaderContainer,
	)

	c := container.NewBorder(top, bottom,
		nil, nil,
		table)

	return widget.NewSimpleRenderer(c)
}
