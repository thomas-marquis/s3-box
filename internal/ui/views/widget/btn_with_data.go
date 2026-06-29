package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type ButtonWithData struct {
	widget.Button

	data binding.String
}

func NewButtonWithData(data binding.String, onTapped func()) *ButtonWithData {
	w := &ButtonWithData{
		data: data,
	}
	w.ExtendBaseWidget(w)
	w.OnTapped = onTapped
	val, err := data.Get()
	if err == nil {
		w.SetText(val)
	}
	data.AddListener(binding.NewDataListener(func() {
		val, _ := data.Get()
		w.SetText(val)
	}))
	return w
}

func (w *ButtonWithData) CreateRenderer() fyne.WidgetRenderer {
	return w.Button.CreateRenderer()
}
