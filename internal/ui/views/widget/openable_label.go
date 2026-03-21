package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	fyne_widget "fyne.io/fyne/v2/widget"
)

type OpenableLabel struct {
	*fyne_widget.Label

	window fyne.Window
}

var (
	_ fyne.Widget         = (*OpenableLabel)(nil)
	_ fyne.DoubleTappable = (*OpenableLabel)(nil)
	_ fyne.Tappable       = (*OpenableLabel)(nil)
	_ desktop.Cursorable  = (*OpenableLabel)(nil)
)

func NewOpenableLabel(text string, window fyne.Window) *OpenableLabel {
	l := &OpenableLabel{fyne_widget.NewLabel(text), window}
	l.ExtendBaseWidget(l)
	return l
}

func (l *OpenableLabel) DoubleTapped(*fyne.PointEvent) {
	l.showDialog()
}

func (l *OpenableLabel) Tapped(*fyne.PointEvent) {
	l.showDialog()
}

func (l *OpenableLabel) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (l *OpenableLabel) showDialog() {
	content := fyne_widget.NewLabel(l.Text)
	content.Wrapping = fyne.TextWrapWord
	content.Alignment = fyne.TextAlignLeading
	content.Selectable = true

	d := dialog.NewCustom("", "Ok", content, l.window)
	d.Resize(fyne.NewSize(600, 300))
	d.Show()
}
