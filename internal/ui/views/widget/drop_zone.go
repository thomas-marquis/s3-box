package widget

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type DropZone struct {
	widget.BaseWidget

	Text string
	Icon fyne.Resource

	OnFilesDropped func(uris []fyne.URI)
	OnClick        func(dropped bool)

	renderer *dropZoneRenderer

	originalText string

	dropAnim         *fyne.Animation
	dropped, hovered bool
	window           fyne.Window
	cancelBtn        *widget.Button
	background       *canvas.Rectangle
}

func NewDropZone(text string, win fyne.Window) *DropZone {
	w := &DropZone{
		Text:           text,
		originalText:   text,
		OnFilesDropped: func(uris []fyne.URI) {},
		window:         win,
		cancelBtn:      widget.NewButton("Cancel", func() {}),
	}
	w.ExtendBaseWidget(w)

	win.SetOnDropped(func(position fyne.Position, uris []fyne.URI) {
		dzSize := w.Size()
		d := fyne.CurrentApp().Driver()
		dzAbsPos := d.AbsolutePositionForObject(w)

		if position.X > dzAbsPos.X && position.X < dzAbsPos.X+dzSize.Width && position.Y > dzAbsPos.Y && position.Y < dzAbsPos.Y+dzSize.Height {
			fmt.Println("DZ is in the drop zone")
			w.dropAnimation()
			if w.OnFilesDropped != nil {
				w.OnFilesDropped(uris)
			}
			w.dropped = true
		}
	})

	return w
}

func (w *DropZone) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	w.cancelBtn.Hide()

	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	bg := canvas.NewRectangle(th.Color(theme.ColorNameBackground, v))
	bg.FillColor = color.Transparent
	bg.CornerRadius = th.Size(theme.SizeNameInputRadius) + 2
	bg.StrokeColor = th.Color(theme.ColorNamePrimary, v)
	bg.StrokeWidth = 1

	w.background = bg

	seg := &widget.TextSegment{Text: w.Text, Style: widget.RichTextStyleStrong}
	seg.Style.Alignment = fyne.TextAlignCenter
	text := widget.NewRichText(seg)

	w.dropAnim = fyne.NewAnimation(
		canvas.DurationStandard,
		func(done float32) {
			if done == 1 {
				bg.FillColor = th.Color(theme.ColorNamePressed, v)
				return
			}
			r, g, bb, a := uiutils.ToNRGBA(th.Color(theme.ColorNamePressed, v))
			aa := uint8(a)
			fade := aa - uint8(float32(aa)*done)
			if fade > 0 {
				bg.FillColor = &color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(bb), A: fade}
			} else {
				bg.FillColor = color.Transparent
			}
		},
	)
	w.dropAnim.Curve = fyne.AnimationEaseOut
	objects := []fyne.CanvasObject{
		bg,
		text,
		w.cancelBtn,
	}

	r := &dropZoneRenderer{
		BaseRenderer: uiutils.NewBaseRenderer(objects),
		w:            w,
		text:         text,
		background:   bg,
	}

	r.text.Refresh()
	r.background.Refresh()

	w.renderer = r

	return r
}

func (w *DropZone) Reset() {
	w.dropped = false
	w.Text = w.originalText

	if w.background != nil {
		w.background.FillColor = color.Transparent
	}
	w.Refresh()
}

func (w *DropZone) Cursor() desktop.Cursor {
	if w.dropped {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

// MouseIn is called when a desktop pointer enters the widget
func (w *DropZone) MouseIn(*desktop.MouseEvent) {
	w.hovered = true
	w.Refresh()
}

// MouseMoved is called when a desktop pointer hovers over the widget
func (w *DropZone) MouseMoved(*desktop.MouseEvent) {
}

// MouseOut is called when a desktop pointer exits the widget
func (w *DropZone) MouseOut() {
	w.hovered = false
	w.Refresh()
}

func (w *DropZone) Tapped(e *fyne.PointEvent) {
	if w.OnClick != nil {
		w.OnClick(w.dropped)
	}
}

func (w *DropZone) dropAnimation() {
	if w.dropAnim == nil {
		return
	}
	w.dropAnim.Stop()
	if fyne.CurrentApp().Settings().ShowAnimations() {
		w.dropAnim.Start()
	}
}

type dropZoneRenderer struct {
	uiutils.BaseRenderer

	text       *widget.RichText
	background *canvas.Rectangle
	w          *DropZone
}

func (r *dropZoneRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)

	textSize := r.text.MinSize()

	textPos := fyne.NewPos(
		(size.Width-textSize.Width)/2,
		(size.Height-textSize.Height)/2,
	)
	r.text.Move(textPos)
	r.text.Resize(textSize)
}

func (r *dropZoneRenderer) MinSize() fyne.Size {
	var size fyne.Size
	th := r.w.Theme()
	size.Width = r.text.MinSize().Width
	size.Height = r.text.MinSize().Height + 100
	size.Add(fyne.NewSquareSize(th.Size(theme.SizeNameInnerPadding) * 2))
	return size
}

func (r *dropZoneRenderer) Refresh() {
	r.text.Segments[0].(*widget.TextSegment).Text = r.w.Text

	th := r.w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if r.w.dropped {
		r.background.FillColor = th.Color(theme.ColorNamePressed, v)
	} else if r.w.hovered {
		r.background.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		r.background.FillColor = color.Transparent
	}

	r.text.Refresh()
	r.background.Refresh()
	r.Layout(r.w.Size())
	canvas.Refresh(r.w)
}
