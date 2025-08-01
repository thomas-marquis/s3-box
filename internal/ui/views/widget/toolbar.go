package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ToolbarButton struct {
	Text string
	Icon fyne.Resource

	onTapped func()
	button   *widget.Button
}

func NewToolbarButton(text string, icon fyne.Resource, onTapped func()) *ToolbarButton {
	return &ToolbarButton{
		Text:     text,
		Icon:     icon,
		onTapped: onTapped,
		button:   widget.NewButtonWithIcon(text, icon, onTapped),
	}
}

func (t *ToolbarButton) ToolbarObject() fyne.CanvasObject {
	return t.button
}

func (t *ToolbarButton) SetOnTapped(f func()) {
	t.button.OnTapped = f
}

func (t *ToolbarButton) Disable() {
	t.button.Disable()
}

func (t *ToolbarButton) Enable() {
	t.button.Enable()
}
