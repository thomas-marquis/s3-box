package imgviewer

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/u"
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

	e.Err.AddListener(binding.NewDataListener(func() {
		err, _ := e.Err.Get()
		if err == nil {
			return
		}
		dialog.ShowError(err, e.Window())
		u.Skip(e.Err.Set(nil))
	}))

	e.ConfirmClose = func(onConfirm func(confirmed bool)) {
		onConfirm(true)
	}

	return w
}

func (w *ImageViewerWidget) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	// Create the filename label
	filenameLabel := widget.NewLabel(w.editor.File().Name().String())
	filenameLabel.Alignment = fyne.TextAlignCenter

	// Create the loader
	loader := widget.NewProgressBarInfinite()
	loader.Stop()

	// Create cancel button for loading
	var cancelBtn *widget.Button
	cancelBtn = widget.NewButton("Cancel", func() {
		cancelBtn.Disable()
		u.Skip(w.editor.StatusLabel.Set("cancelling..."))
		w.editor.Cancel()
	})
	cancelBtn.Hide()

	loaderContainer := container.NewBorder(
		nil, nil, nil,
		cancelBtn, loader,
	)

	// Create the image display
	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillOriginal

	// Create scroll container for the image
	scrollContainer := container.NewScroll(img)
	scrollContainer.Hide()

	// Handle loading state changes
	w.editor.IsLoading.AddListener(binding.NewDataListener(func() {
		isLoading, _ := w.editor.IsLoading.Get()
		if isLoading {
			loaderContainer.Show()
			loader.Start()
			scrollContainer.Hide()
			cancelBtn.Show()
		} else {
			loaderContainer.Hide()
			loader.Stop()
			cancelBtn.Hide()
		}
	}))

	// Handle image data changes
	w.editor.ImageData.AddListener(binding.NewDataListener(func() {
		imageData, err := w.editor.ImageData.Get()
		if err != nil || imageData == nil {
			scrollContainer.Hide()
			return
		}

		img.Resource = fyne.NewStaticResource(w.editor.File().Name().String(), imageData)
		img.Refresh()
		scrollContainer.Show()
	}))

	// Initial state setup
	if u.SkipV(w.editor.IsLoading.Get()) {
		loader.Start()
		loaderContainer.Show()
		cancelBtn.Show()
	} else {
		// Check if we already have image data
		imageData, _ := w.editor.ImageData.Get()
		if imageData != nil {
			img.Resource = fyne.NewStaticResource(w.editor.File().Name().String(), imageData)
			scrollContainer.Show()
		} else {
			loaderContainer.Hide()
		}
	}

	// Main layout
	c := container.NewBorder(
		container.NewBorder(nil, nil,
			widget.NewLabelWithData(w.editor.StatusLabel),
			nil,
			filenameLabel),
		loaderContainer,
		nil, nil,
		scrollContainer,
	)

	return widget.NewSimpleRenderer(c)
}
