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

// CellEntry is a custom entry widget that can be bound to a table cell.
// It extends widget.Entry to add binding support and custom shortcuts.
type CellEntry struct {
	widget.Entry
	row, col   int
	val        binding.String
	IsReadOnly bool

	OnClose, OnSave func()

	dl binding.DataListener
}

func newCellEntry() *CellEntry {
	e := &CellEntry{}
	e.Validator = nil
	e.ExtendBaseWidget(e)
	return e
}

// Bind binds the cell entry to a string binding.
// The cell will display the binding's value and update it when edited.
func (e *CellEntry) Bind(data binding.String) {
	th := e.Theme()
	textSize := th.Size(theme.SizeNameText)

	// Clean up previous binding
	if e.dl != nil && e.val != nil {
		e.val.RemoveListener(e.dl)
	}

	e.val = data
	e.dl = binding.NewDataListener(func() {
		text, err := data.Get()
		if err != nil {
			return
		}
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
	}
}

func (e *CellEntry) TypedRune(r rune) {
	if e.IsReadOnly {
		return
	}
	e.Entry.TypedRune(r)
}

// UpdateCoords updates the cell's coordinate tracking.
func (e *CellEntry) UpdateCoords(row, col int) {
	e.row = row
	e.col = col
}
