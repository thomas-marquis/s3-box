package explorerview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type treeItem struct{}

func newTreeItemBuilder() *treeItem {
	return &treeItem{}
}

func (i *treeItem) NewRaw() *fyne.Container {
	name := widget.NewLabel(".")
	return container.NewHBox(name)
}

func (i *treeItem) Update(o fyne.CanvasObject, contentName string) {
	c, _ := o.(*fyne.Container)
	contentLabel := c.Objects[0].(*widget.Label)
	contentLabel.SetText(contentName)
}
