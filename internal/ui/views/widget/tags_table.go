package widget

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
)

const (
	tagsWidgetObserversName = "tags.observers.widget"
	tableColKey             = 0
	tableColValue           = 1
	tableColActions         = 2
)

// TagsTable is a singleton widget that displays an editable and savable list of tags.
type TagsTable struct {
	widget.BaseWidget
	mu     sync.Mutex
	AddBtn *widget.Button
	vm     viewmodel.TagsViewmodel
	state  *state.State
	appCtx appcontext.AppContext
	table  *widget.Table

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

	w.AddBtn = widget.NewButton("New tag", func() {
		dataKey := binding.NewString()
		dataValue := binding.NewString()

		dial := w.makeEditFormDialog("New tag", dataKey, dataValue, func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := w.vm.Add(u.SkipV(dataKey.Get()), u.SkipV(dataValue.Get())); err != nil {
				u.Skip(w.state.Tags().StatusLabel().Set(
					fmt.Sprintf("failed to add tag: %s", err.Error())))
			}

			u.Skip(dataKey.Set(""))
			u.Skip(dataValue.Set(""))
		})
		dial.Show()
	})

	w.table = widget.NewTableWithHeaders(
		func() (rows, cols int) {
			return w.state.Tags().DisplayedTags().Length(), 3
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Selectable = true

			delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
			delBtn.Importance = widget.LowImportance

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {})
			editBtn.Importance = widget.LowImportance

			rowActions := container.NewHBox(
				delBtn,
				editBtn,
			)
			rowActions.Hide()

			c := container.NewStack(label, rowActions)
			return c
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			label := c.Objects[0].(*widget.Label)
			actions := c.Objects[1].(*fyne.Container)

			delBtn := actions.Objects[0].(*widget.Button)
			editBtn := actions.Objects[1].(*widget.Button)

			ts := w.state.Tags().CurrentTagSet()
			if ts == nil {
				return
			}

			tagDataItem, err := w.state.Tags().DisplayedTags().GetItem(id.Row)
			if err != nil {
				return
			}
			boundTag := tagDataItem.(binding.Item[directory.Tag])

			switch id.Col {
			case tableColKey:
				label.Show()
				actions.Hide()
				label.Bind(w.makeTagKeyBinding(boundTag))

			case tableColValue:
				label.Show()
				actions.Hide()
				label.Bind(w.makeTagValueBinding(boundTag))

			case tableColActions:
				label.Hide()
				actions.Show()

				delBtn.OnTapped = w.makeOnTagDelete(boundTag)
				editBtn.OnTapped = w.makeOnTagEdit(ts, boundTag)
			}
		},
	)

	w.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	w.table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		switch id.Col {
		case tableColKey:
			template.(*widget.Label).SetText("Key")
		case tableColValue:
			template.(*widget.Label).SetText("Value")
		}
	}
	w.table.ShowHeaderColumn = false

	w.state.Tags().DisplayedTags().AddListener(binding.NewDataListener(w.table.Refresh))

	return w
}

func (w *TagsTable) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	w.table.SetColumnWidth(0, 150)
	w.table.SetColumnWidth(1, 200)
	w.table.HideSeparators = true

	c := container.NewBorder(
		container.NewBorder(w.noTagsLabel, nil,
			container.NewHBox(
				w.AddBtn,
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

func (w *TagsTable) makeEditFormDialog(title string, keyData, valueData binding.String, cb func(bool)) *dialog.FormDialog {
	keyEntry := widget.NewEntryWithData(keyData)
	valueEntry := widget.NewEntryWithData(valueData)

	d := dialog.NewForm(title, "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Key", keyEntry),
		widget.NewFormItem("Value", valueEntry),
	}, cb, w.appCtx.Window())
	d.Resize(fyne.NewSize(350, 100))

	return d
}

func (w *TagsTable) makeOnTagEdit(ts *directory.TagSet, boundTag binding.Item[directory.Tag]) func() {
	return func() {
		tag := u.SkipV(boundTag.Get())

		dataKey := binding.NewString()
		u.Skip(dataKey.Set(tag.Key))

		dataValue := binding.NewString()
		u.Skip(dataValue.Set(tag.Value))

		dial := w.makeEditFormDialog("Update tag", dataKey, dataValue, func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := ts.Update(tag.Key, u.SkipV(dataKey.Get()), u.SkipV(dataValue.Get())); err != nil {
				u.Skip(w.state.Tags().StatusLabel().Set(
					fmt.Sprintf("failed to update tag: %s", err.Error())))
			}
			w.vm.Save()
		})
		dial.Show()
	}
}

func (w *TagsTable) makeOnTagDelete(boundTag binding.Item[directory.Tag]) func() {
	return func() {
		tag := u.SkipV(boundTag.Get())
		dialog.ShowConfirm("Delete tag",
			fmt.Sprintf("Your tag '%s' will be deleted. Sure?", tag.Key),
			func(sure bool) {
				if !sure {
					return
				}
				w.vm.Delete(tag.Key)
			}, w.appCtx.Window())
	}
}

func (w *TagsTable) makeTagKeyBinding(boundTag binding.Item[directory.Tag]) binding.String {
	return uu.NewBindMapper[directory.Tag, string](boundTag,
		func(tag directory.Tag) string {
			return tag.Key
		},
		func(key string) directory.Tag {
			displayedTag := u.SkipV(boundTag.Get())
			return directory.Tag{Key: key, Value: displayedTag.Value}
		},
		func(tag directory.Tag, key string) bool {
			return tag.Key == key
		},
	)
}

func (w *TagsTable) makeTagValueBinding(boundTag binding.Item[directory.Tag]) binding.String {
	return uu.NewBindMapper[directory.Tag, string](boundTag,
		func(tag directory.Tag) string {
			return tag.Value
		},
		func(value string) directory.Tag {
			displayedTag := u.SkipV(boundTag.Get())
			return directory.Tag{Value: value, Key: displayedTag.Key}
		},
		func(tag directory.Tag, value string) bool {
			return tag.Value == value
		},
	)
}
