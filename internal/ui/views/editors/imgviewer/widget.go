package imgviewer

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ImageViewerWidget struct {
	widget.BaseWidget
	editor *Editor
}

func newWidget(e *Editor) fyne.CanvasObject {
	w := &ImageViewerWidget{
		editor: e,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ImageViewerWidget) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromReader(r, "")
	img.FillMode = canvas.ImageFillOriginal

	c := container.NewBorder(nil, nil, nil, nil, img)
	return widget.NewSimpleRenderer(c)
}
