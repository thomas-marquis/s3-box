package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
)

type TreeItem struct {
}

func NewTreeItemBuilder() *TreeItem {
	return &TreeItem{}
}

func (i *TreeItem) NewRaw() *fyne.Container {
	displayLabel := widget.NewLabel("-")
	icon := widget.NewIcon(theme.FolderIcon())
	icon.Hide()
	return container.NewHBox(icon, displayLabel)
}

func (i *TreeItem) Update(o fyne.CanvasObject, nodeItem node.Node) {
	c, _ := o.(*fyne.Container)
	icon := c.Objects[0].(*widget.Icon)
	displayLabel := c.Objects[1].(*widget.Label)

	displayLabel.SetText(nodeItem.DisplayName())

	if nodeItem.Icon != nil {
		icon.SetResource(nodeItem.Icon())
		icon.Show()
	} else {
		icon.Hide()
	}
}
