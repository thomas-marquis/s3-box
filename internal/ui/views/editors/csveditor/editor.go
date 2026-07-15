package csveditor

import (
	"encoding/csv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

type csvColumn struct {
	Width float32
}

type csvEditor struct {
	editor.Base

	bus event.Bus

	Records binding.List[[]string]
	Columns binding.List[csvColumn]
}

func New(bus event.Bus, w fyne.Window, file *directory.File) editor.Editor {
	ed := &csvEditor{
		Base: editor.NewBase(w, file),
		bus:  bus,
		Records: binding.NewList[[]string](func(l1, l2 []string) bool {
			if len(l1) != len(l2) {
				return false
			}
			for i := range l1 {
				if l1[i] != l2[i] {
					return false
				}
			}
			return true
		}),
		Columns: binding.NewList[csvColumn](func(c1, c2 csvColumn) bool {
			return c1 == c2
		}),
	}

	return ed
}

func (e *csvEditor) CreateWidget() fyne.CanvasObject {
	return newWidget(e)
}

func (e *csvEditor) OnLoaded(fileContent directory.FileContent, err error) {
	if err != nil {

		// TODO
		return
	}

	r := csv.NewReader(fileContent)

	nbRows := 0
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		e.Records.Append(record) //nolint:errcheck
		nbRows++
	}

	if e.Records.Length() == 0 {
		return
	}

	th := fyne.CurrentApp().Settings().Theme()
	textSize := th.Size(theme.SizeNameText)

	firstRow, _ := e.Records.GetValue(0)
	for i := range len(firstRow) {
		col := csvColumn{}
		for j := range nbRows {
			row, _ := e.Records.GetValue(j)
			ts := fyne.MeasureText(row[i], textSize, fyne.TextStyle{})
			if col.Width < ts.Width {
				col.Width = ts.Width
			}
		}
		e.Columns.Append(col) //nolint:errcheck
	}
}

func (e *csvEditor) OnSaved(newContent string, err error) {
	//TODO implement me
	panic("implement me")
}

func (e *csvEditor) Close() bool {
	return true
}
