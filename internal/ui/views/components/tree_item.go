package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type TreeItem struct{}

func NewTreeItemBuilder() *TreeItem {
	return &TreeItem{}
}

func (i *TreeItem) NewRaw() *fyne.Container {
	name := widget.NewLabel(".")
	return container.NewHBox(name)
}

func (i *TreeItem) Update(o fyne.CanvasObject, contentName string) {
	c, _ := o.(*fyne.Container)
	contentLabel := c.Objects[0].(*widget.Label)
	contentLabel.SetText(contentName)
}
