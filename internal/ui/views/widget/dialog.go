package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type ActionShortcuts struct {
	Shortcuts []desktop.CustomShortcut
	Callback  func()
}

type EntryWithShortcuts struct {
	widget.Entry

	Actions []ActionShortcuts
}

var (
	_ fyne.Shortcutable = (*EntryWithShortcuts)(nil)
)

func NewEntryWithShortcuts(actions []ActionShortcuts) *EntryWithShortcuts {
	w := &EntryWithShortcuts{
		Actions: actions,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (d *EntryWithShortcuts) TypedShortcut(s fyne.Shortcut) {
	val, ok := s.(*desktop.CustomShortcut)
	if !ok {
		d.TypedShortcut(s)
		return
	}
	for _, a := range d.Actions {
		for _, v := range a.Shortcuts {
			if val.KeyName == v.Key() && val.Modifier == v.Mod() {
				a.Callback()
				return
			}
		}
	}
}
