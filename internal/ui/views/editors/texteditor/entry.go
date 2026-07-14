package texteditor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type textContentEntry struct {
	widget.Entry

	onValidate func(string)
	onClose    func()
}

var (
	_ fyne.Shortcutable = (*textContentEntry)(nil)
)

func (e *textContentEntry) TypedShortcut(s fyne.Shortcut) {
	if val, ok := s.(*desktop.CustomShortcut); ok {
		if val.KeyName == fyne.KeyS && val.Modifier == fyne.KeyModifierControl {
			e.onValidate(e.Text)
		} else if val.KeyName == fyne.KeyQ && val.Modifier == fyne.KeyModifierControl {
			e.onClose()
		}
	} else {
		e.Entry.TypedShortcut(s)
	}
}

func newTextEditorEntry(onValidate func(string), onCLose func()) *textContentEntry {
	e := &textContentEntry{
		Entry: widget.Entry{
			MultiLine: true,
			Wrapping:  fyne.TextWrap(fyne.TextTruncateClip),
		},
		onValidate: onValidate,
		onClose:    onCLose,
	}
	e.ExtendBaseWidget(e)
	return e
}
