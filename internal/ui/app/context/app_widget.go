package appcontext

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
)

type AppWidget struct {
	widget.BaseWidget

	menu  []Menu
	navCb func(navigation.Route) (*fyne.Container, error)
	split *container.Split
}

var _ fyne.Widget = (*AppWidget)(nil)

func newAppWidget(menus []Menu, navCb func(navigation.Route) (*fyne.Container, error)) *AppWidget {
	a := &AppWidget{
		menu:  menus,
		navCb: navCb,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *AppWidget) CreateRenderer() fyne.WidgetRenderer {
	a.ExtendBaseWidget(a)

	var content fyne.CanvasObject
	var sMax float32
	btns := make([]fyne.CanvasObject, len(a.menu))
	for i, m := range a.menu {
		b := widget.NewButtonWithIcon(m.Label, m.IconFactory(), func() {
			view, err := a.navCb(m.Route)
			if err != nil {
				return
			}
			content = view
		})
		if s := b.MinSize().Width; s > sMax {
			sMax = s
		}
		b.Alignment = widget.ButtonAlignLeading
		btns[i] = b
	}
	for i := range btns {
		btns[i].Resize(fyne.NewSize(sMax, btns[i].MinSize().Height))
	}
	itemList := container.NewVBox(btns...)

	content = widget.NewLabel("")
	split := container.NewHSplit(itemList, content)
	split.SetOffset(0)

	a.split = split

	return widget.NewSimpleRenderer(split)
}

func (a *AppWidget) SetViewContent(vc fyne.CanvasObject) {
	if a.split == nil {
		return
	}
	a.split.Trailing = vc
	a.split.Refresh()
}
