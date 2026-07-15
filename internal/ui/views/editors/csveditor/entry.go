package csveditor

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	errOutOfBounds = errors.New("out of bounds")
)

type CellEntry struct {
	widget.Entry
	records  binding.List[[]string]
	row, col int
	val      binding.String

	OnClose, OnSave func()
}

func newCellEntry(records binding.List[[]string]) *CellEntry {
	val := binding.NewString()
	e := &CellEntry{
		records: records,
		val:     val,
	}

	th := e.Theme()
	textSize := th.Size(theme.SizeNameText)

	val.AddListener(binding.NewDataListener(func() {
		text, err := val.Get()
		if err != nil {
			return
		}
		if err := e.updateRecord(text); err != nil {
			return
		}
		currWidth := e.Size().Width
		textWidth := colWidth(text, textSize)
		if textWidth > currWidth {
			e.Scroll = fyne.ScrollHorizontalOnly
		} else {
			e.Scroll = fyne.ScrollNone
		}
	}))
	e.Bind(val)

	e.Validator = nil

	e.ExtendBaseWidget(e)
	return e
}

func (e *CellEntry) TypedShortcut(s fyne.Shortcut) {
	if sc, ok := s.(*desktop.CustomShortcut); ok {
		if e.OnSave != nil && *sc == shortcutSave {
			e.OnSave()
		} else if e.OnClose != nil && *sc == shortcutQuit {
			e.OnClose()
		}
	}
}

func (e *CellEntry) UpdateCoords(row, col int) {
	e.row = row
	e.col = col
}

//func (e *CellEntry) Value() (string, error) {
//	val, err := e.records.GetValue(e.row)
//	if err != nil {
//		return "", err
//	}
//	if len(val) <= e.col {
//		return "", errOutOfBounds
//	}
//
//	return val[e.col], nil
//}

func (e *CellEntry) updateRecord(text string) error {
	row, err := e.records.GetValue(e.row)
	if err != nil {
		return err
	}
	if len(row) <= e.col {
		return errOutOfBounds
	}

	row[e.col] = text
	return nil
}
