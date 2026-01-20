package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Heading struct {
	widget.BaseWidget

	Text string
}

var _ fyne.Widget = (*Heading)(nil)

func NewHeading(text string) *Heading {
	h := &Heading{Text: text}
	h.ExtendBaseWidget(h)
	return h
}

func (h *Heading) CreateRenderer() fyne.WidgetRenderer {
	h.ExtendBaseWidget(h)
	seg := &widget.TextSegment{
		Text: h.Text,
		Style: widget.RichTextStyle{
			ColorName: theme.ColorNameForeground,
			TextStyle: fyne.TextStyle{Bold: false},
			SizeName:  theme.SizeNameHeadingText,
		},
	}
	rt := widget.NewRichText(seg)

	return widget.NewSimpleRenderer(rt)
}
