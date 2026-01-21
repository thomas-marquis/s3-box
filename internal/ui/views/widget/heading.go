package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Heading struct {
	widget.BaseWidget

	Text string

	data        binding.String
	richText    *widget.RichText
	textSegment *widget.TextSegment
}

var _ fyne.Widget = (*Heading)(nil)

func NewHeading(text string) *Heading {
	h := &Heading{Text: text}
	h.ExtendBaseWidget(h)

	h.textSegment = &widget.TextSegment{
		Text: h.Text,
		Style: widget.RichTextStyle{
			ColorName: theme.ColorNameForeground,
			TextStyle: fyne.TextStyle{Bold: false},
			SizeName:  theme.SizeNameHeadingText,
		},
	}
	h.richText = widget.NewRichText(h.textSegment)

	return h
}

func NewHeadingWithData(data binding.String) *Heading {
	h := NewHeading("")

	h.data = data
	if h.data != nil {
		h.data.AddListener(binding.NewDataListener(func() {
			t, _ := h.data.Get()
			h.textSegment.Text = t
			h.Refresh()
		}))
	}

	return h
}

func (h *Heading) CreateRenderer() fyne.WidgetRenderer {
	h.ExtendBaseWidget(h)
	return widget.NewSimpleRenderer(h.richText)
}
