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
	shortcutQuit   = desktop.CustomShortcut{
		KeyName:  fyne.KeyQ,
		Modifier: fyne.KeyModifierControl,
	}
)

type CellEntry struct {
	widget.Entry
	row, col   int
	val        binding.String
	IsReadOnly bool

	OnClose, OnSave func()

	dl binding.DataListener
}

func newCellEntry() *CellEntry {
	// val := binding.NewString()
	e := &CellEntry{
		// val: val,
	}

	// th := e.Theme()
	// textSize := th.Size(theme.SizeNameText)

	// var initialized atomic.Bool
	// initialized.Store(false)

	// val.AddListener(binding.NewDataListener(func() {
	// 	if !initialized.Load() {
	// 		initialized.Store(true)
	// 		return
	// 	}

	// 	text, err := val.Get()
	// 	if err != nil {
	// 		return
	// 	}
	// 	if err := e.updateRecord(text); err != nil {
	// 		return
	// 	}
	// 	currWidth := e.Size().Width
	// 	textWidth := colWidth(text, textSize)
	// 	if textWidth > currWidth {
	// 		e.Scroll = fyne.ScrollHorizontalOnly
	// 	} else {
	// 		e.Scroll = fyne.ScrollNone
	// 	}
	// }))
	// e.Bind(val)

	e.Validator = nil

	e.ExtendBaseWidget(e)
	return e
}

func (e *CellEntry) Bind(data binding.String) {
	th := e.Theme()
	textSize := th.Size(theme.SizeNameText)

	if e.dl != nil {
		data.RemoveListener(e.dl)
	}

	e.dl = binding.NewDataListener(func() {
		// if !initialized.Load() {
		// 	initialized.Store(true)
		// 	return
		// }

		text, err := data.Get()
		if err != nil {
			return
		}
		// if err := e.updateRecord(text); err != nil {
		// 	return
		// }
		currWidth := e.Size().Width
		textWidth := colWidth(text, textSize)
		if textWidth > currWidth {
			e.Scroll = fyne.ScrollHorizontalOnly
		} else {
			e.Scroll = fyne.ScrollNone
		}
	})

	data.AddListener(e.dl)
}

func (e *CellEntry) TypedShortcut(s fyne.Shortcut) {
	if sc, ok := s.(*desktop.CustomShortcut); ok {
		if e.OnSave != nil && !e.IsReadOnly && *sc == shortcutSave {
			e.OnSave()
		} else if e.OnClose != nil && *sc == shortcutQuit {
			e.OnClose()
		}

		// TODO: disable paste shortcut when read only is true
	}
}

func (e *CellEntry) TypedRune(r rune) {
	if e.IsReadOnly {
		return
	}
	e.Entry.TypedRune(r)
}

func (e *CellEntry) UpdateCoords(row, col int) {
	e.row = row
	e.col = col
}

// func (e *CellEntry) updateRecord(text string) error {
// 	row, err := e.records.GetValue(e.row)
// 	if err != nil {
// 		return err
// 	}
// 	if len(row) <= e.col {
// 		return errOutOfBounds
// 	}

// 	row[e.col] = text
// 	return nil
// }
