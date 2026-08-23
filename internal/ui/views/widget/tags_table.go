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

	noTagsLabel   *widget.Label
	noTagsBinding binding.String
}

func NewTagsTable(appCtx appcontext.AppContext, vm viewmodel.TagsViewmodel) *TagsTable {
	w := &TagsTable{
		vm:            vm,
		appCtx:        appCtx,
		state:         appCtx.State(),
		noTagsBinding: binding.NewString(),
	}

	w.ExtendBaseWidget(w)

	w.noTagsLabel = widget.NewLabelWithData(w.noTagsBinding)
	w.noTagsLabel.Hide()

	w.SaveBtn = widget.NewButton("Save", func() {
		w.vm.Save()
		w.SaveBtn.Disable()
	})

	w.AddBtn = widget.NewButton("New tag", func() {
		dataKey := binding.NewString()
		dataValue := binding.NewString()

		keyEntry := widget.NewEntryWithData(dataKey)
		keyEntry.Resize(fyne.NewSize(150, keyEntry.Size().Height))

		valEntry := widget.NewEntryWithData(dataValue)
		valEntry.Resize(fyne.NewSize(150, valEntry.Size().Height))

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
			entry := NewTextEntry()
			entry.Validator = func(string) error {
				return nil
			}
			return entry
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			entry := o.(*TextEntry)

			ts := w.state.Tags().CurrentTagSet()
			if ts == nil {
				return
			}

			tags := ts.Get()
			tag := tags[id.Row]

			if id.Col == 0 {
				// Key
				entry.SetText(tag.Key)
				entry.OnTyped = func(newKey string) {
					entry.SetValidationError(ts.Update(tag.Key, newKey, tag.Value))
					w.SaveBtn.Show()
					w.SaveBtn.Enable()
				}
			} else {
				// Value
				entry.SetText(tag.Value)
				entry.OnTyped = func(newValue string) {
					entry.SetValidationError(ts.Update(tag.Key, tag.Key, newValue))
					w.SaveBtn.Show()
					w.SaveBtn.Enable()
				}
			}
		},
	)

	return w
}

func (w *TagsTable) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	w.SaveBtn.Hide()

	w.table.SetColumnWidth(0, 150)
	w.table.SetColumnWidth(1, 200)
	w.table.HideSeparators = true

	c := container.NewBorder(
		container.NewBorder(w.noTagsLabel, nil,
			container.NewHBox(
				w.AddBtn,
				w.SaveBtn,
			),
			widget.NewLabelWithData(w.state.Tags().StatusLabel()),
		),
		nil, nil, nil,
		w.table,
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

	ts.ObserveWithName(tagsWidgetObserversName, func(tags []directory.Tag) {
		fyne.Do(func() {
			if len(tags) == 0 {
				w.table.Hide()
				u.Skip(w.noTagsBinding.Set("No tags to display"))
				w.noTagsLabel.Show()
			} else {
				w.table.Show()
				u.Skip(w.noTagsBinding.Set(""))
				w.noTagsLabel.Hide()
			}
		})
	})
	ts.TriggerWithName(tagsWidgetObserversName, ts.Get())

	w.vm.Select(ts)
}
