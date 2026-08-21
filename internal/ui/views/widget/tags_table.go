package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type TagsTable struct {
	widget.BaseWidget
}

func NewTagsTable() *TagsTable {
	w := &TagsTable{}

	w.ExtendBaseWidget(w)
	return w
}

func (w *TagsTable) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	tags := binding.NewList[directory.Tag](directory.CompareTag)
	table := widget.NewListWithData(
		tags,
		func() fyne.CanvasObject {
			key := widget.NewEntry()
			value := widget.NewEntry()

			return container.NewHBox(
				key, value,
			)
		},
		func(it binding.DataItem, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			keyEntry := c.Objects[0].(*widget.Entry)
			valueEntry := c.Objects[1].(*widget.Entry)

			tag, err := it.(binding.Item[directory.Tag]).Get()
			if err != nil {
				panic(fmt.Errorf("unexpected type %T: %w", tag, err))
			}

			keyEntry.SetText(tag.Key)
			valueEntry.SetText(tag.Value)
		},
	)

	return widget.NewSimpleRenderer(table)
}
