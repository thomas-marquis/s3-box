package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	fyne_widget "fyne.io/fyne/v2/widget"
)

type OpenableLabel struct {
	fyne_widget.BaseWidget

	Label  *fyne_widget.Label
	Detail *fyne_widget.Label

	window fyne.Window
}

var (
	_ fyne.Widget         = (*OpenableLabel)(nil)
	_ fyne.DoubleTappable = (*OpenableLabel)(nil)
	_ fyne.Tappable       = (*OpenableLabel)(nil)
	_ desktop.Cursorable  = (*OpenableLabel)(nil)
)

func NewOpenableLabel(text string, window fyne.Window) *OpenableLabel {
	l := &OpenableLabel{
		Label:  fyne_widget.NewLabel(text),
		Detail: fyne_widget.NewLabel(""),
		window: window,
	}
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

func (l *OpenableLabel) CreateRenderer() fyne.WidgetRenderer {
	l.ExtendBaseWidget(l)
	return fyne_widget.NewSimpleRenderer(l.Label)
}

func (l *OpenableLabel) showDialog() {
	content := fyne_widget.NewLabel(l.Detail.Text)
	content.Wrapping = fyne.TextWrapWord
	content.Alignment = fyne.TextAlignLeading
	content.Selectable = true

	title := fyne_widget.NewLabel(l.Label.Text)
	title.Selectable = true
	title.TextStyle = fyne.TextStyle{Bold: true}

	c := container.NewVScroll(container.NewVBox(
		title,
		content,
	))

	d := dialog.NewCustom("", "Ok", c, l.window)
	d.Resize(fyne.NewSize(600, 500))
	d.Show()
}
