package widget

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
)

const (
	tagsWidgetObserversName = "tags.observers.widget"
)

// TagsTable is a singleton widget that displays an editable and savable list of tags.
type TagsTable struct {
	widget.BaseWidget
	mu      sync.Mutex
	SaveBtn *widget.Button
	AddBtn  *widget.Button
	vm      viewmodel.TagsViewmodel
	state   *state.State
	appCtx  appcontext.AppContext
	table   *widget.Table
}

func NewTagsTable(appCtx appcontext.AppContext, vm viewmodel.TagsViewmodel) *TagsTable {
	w := &TagsTable{
		vm:     vm,
		appCtx: appCtx,
		state:  appCtx.State(),
	}

	w.ExtendBaseWidget(w)

	w.SaveBtn = widget.NewButton("Save", w.vm.Save)
	w.SaveBtn.Hide()
	return w
}

func (w *TagsTable) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	w.AddBtn = widget.NewButton("New tag", func() {
		dataKey := binding.NewString()
		dataValue := binding.NewString()

		keyEntry := widget.NewEntryWithData(dataKey)
		valEntry := widget.NewEntryWithData(dataValue)

		dialog.ShowForm("New tag", "Add", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Key", keyEntry),
			widget.NewFormItem("Value", valEntry),
		}, func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := w.vm.Add(u.SkipV(dataKey.Get()), u.SkipV(dataValue.Get())); err != nil {
				fyne.LogError("Error adding tag", err)
			}

			u.Skip(dataKey.Set(""))
			u.Skip(dataValue.Set(""))
		}, w.appCtx.Window())
	})

	w.table = widget.NewTable(
		func() (rows, cols int) {
			return w.state.Tags().Length(), 2
		},
		func() fyne.CanvasObject {
			entry := widget.NewEntry()
			entry.Validator = nil
			return entry
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			entry := o.(*widget.Entry)

			ts := w.state.Tags().CurrentTagSet()
			if ts == nil {
				return
			}

			tags := ts.Get()
			tag := tags[id.Row]

			if id.Col == 0 {
				// Key
				entry.SetText(tag.Key)
				entry.OnChanged = func(newKey string) {
					w.SaveBtn.Show()
					entry.SetValidationError(ts.Update(tag.Key, newKey, tag.Value))
				}
			} else {
				// Value
				entry.SetText(tag.Value)
				entry.OnChanged = func(newValue string) {
					w.SaveBtn.Show()
					entry.SetValidationError(ts.Update(tag.Key, tag.Key, newValue))
				}
			}
		},
	)

	c := container.NewVBox(
		w.table,
		container.NewBorder(nil, nil,
			container.NewHBox(
				w.AddBtn,
				w.SaveBtn,
			),
			widget.NewLabelWithData(w.state.Tags().StatusLabel()),
		),
	)

	return widget.NewSimpleRenderer(c)
}

// Select changes the tag set in use for this widget.
func (w *TagsTable) Select(ts *directory.TagSet) {
	w.SaveBtn.Hide()
	prevTs := w.state.Tags().CurrentTagSet()
	if prevTs != nil {
		prevTs.RemoveObserversWithName(tagsWidgetObserversName)
	}

	w.vm.Select(ts)
	ts.ObserveWithName(tagsWidgetObserversName, func([]directory.Tag) {
		if w.table == nil {
			return
		}
		w.table.Refresh()
	})
}
