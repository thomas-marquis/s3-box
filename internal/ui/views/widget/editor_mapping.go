package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/u"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	fileMappingColEditorType = 0
	fileMappingColPattern    = 1
	fileMappingColActions    = 2
)

// EditorMappingsTable is a widget that displays and allows editing of file editor matchers.
type EditorMappingsTable struct {
	widget.BaseWidget
	state  *state.State
	appCtx appcontext.AppContext
	table  *widget.Table

	noMatchersBinding binding.String
}

func NewEditorMappingsTable(appCtx appcontext.AppContext) *EditorMappingsTable {
	w := &EditorMappingsTable{
		appCtx:            appCtx,
		state:             appCtx.State(),
		noMatchersBinding: binding.NewString(),
	}

	w.ExtendBaseWidget(w)

	w.table = widget.NewTableWithHeaders(
		func() (rows, cols int) {
			return len(w.state.Editors().Selector().Mappings()), 3
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Selectable = true

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {})
			editBtn.Importance = widget.LowImportance

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
			editBtn.Importance = widget.LowImportance

			rowActions := container.NewHBox(
				editBtn,
				deleteBtn,
			)
			rowActions.Hide()

			c := container.NewStack(label, rowActions)
			return c
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			label := c.Objects[0].(*widget.Label)
			actions := c.Objects[1].(*fyne.Container)

			editBtn := actions.Objects[0].(*widget.Button)
			deleteBtn := actions.Objects[1].(*widget.Button)

			selector := w.state.Editors().Selector()
			mappings := selector.Mappings()
			if id.Row >= len(mappings) {
				return
			}

			m := mappings[id.Row]

			factory, err := selector.GetRegisteredEditorByName(m.EditorName)
			if err != nil {
				panic(err) // should not happened
			}

			switch id.Col {
			case fileMappingColEditorType:
				label.Show()
				actions.Hide()
				label.SetText(factory.DisplayLabel())

			case fileMappingColPattern:
				label.Show()
				actions.Hide()
				label.SetText(m.RegexpPattern)

			case fileMappingColActions:
				label.SetText("")
				label.Hide()
				actions.Show()

				editBtn.OnTapped = w.makeOnMappingEdit(selector, factory, m)
				deleteBtn.OnTapped = w.makeOnMappingDelete(selector, m)
			}
		},
	)

	w.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	w.table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		switch id.Col {
		case fileMappingColEditorType:
			template.(*widget.Label).SetText("Editor")
		case fileMappingColPattern:
			template.(*widget.Label).SetText("Pattern")
		}
	}
	w.table.ShowHeaderColumn = false

	w.state.Editors().Selector().MappingsObservable().Observe(func([]editor.Mapping) {
		w.table.Refresh()
	})

	return w
}

func (w *EditorMappingsTable) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	w.table.SetColumnWidth(0, 120)
	w.table.SetColumnWidth(1, 300)
	w.table.HideSeparators = true

	addBtn := widget.NewButtonWithIcon("New mapping", theme.ContentAddIcon(), func() {
		patternBinding := binding.NewString()
		selector := w.state.Editors().Selector()
		editorNameBinding := binding.NewString()

		var availableEditorNames []string
		for _, e := range selector.RegisteredEditors() {
			availableEditorNames = append(availableEditorNames, e.Name())
		}

		d := dialog.NewForm("Add a new mapping", "Save", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Editor", widget.NewSelectWithData(availableEditorNames, editorNameBinding)),
			widget.NewFormItem("Pattern (regexp)", widget.NewEntryWithData(patternBinding)),
		}, func(confirmed bool) {
			if !confirmed {
				return
			}

			pattern := u.SkipV(patternBinding.Get())
			editorName := u.SkipV(editorNameBinding.Get())

			if err := selector.RegisterMapping(editorName, pattern); err != nil {
				u.Skip(w.state.Settings().StatusMessage().Set("Failed to add a new mapping"))
				// TODO: send the error to the notifications
				return
			}
			w.appCtx.SettingsViewModel().SaveEditorMappings()
		}, w.appCtx.Window())
		d.Resize(fyne.NewSize(450, 120))
		d.Show()
	})

	c := container.NewBorder(
		widget.NewLabel("Map a file path pattern with a default editor"),
		addBtn,
		nil, nil,
		w.table,
	)

	return widget.NewSimpleRenderer(c)
}

func (w *EditorMappingsTable) Refresh() {
	w.table.Refresh()
}

func (w *EditorMappingsTable) makeEditFormDialog(title string, patternData binding.String, cb func(bool)) *dialog.FormDialog {
	patternEntry := widget.NewEntryWithData(patternData)

	d := dialog.NewForm(title, "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Pattern (regexp)", patternEntry),
	}, cb, w.appCtx.Window())
	d.Resize(fyne.NewSize(450, 120))

	return d
}

func (w *EditorMappingsTable) makeOnMappingEdit(selector *editor.Selector, factory editor.Factory, mapping editor.Mapping) func() {
	return func() {
		pattern := binding.NewString()
		u.Skip(pattern.Set(mapping.RegexpPattern))

		dial := w.makeEditFormDialog(
			fmt.Sprintf("Update pattern for %s editor", factory.DisplayLabel()),
			pattern,
			func(confirmed bool) {
				if !confirmed {
					return
				}

				newPattern := u.SkipV(pattern.Get())
				if err := selector.UpdateMapping(mapping.EditorName, mapping.RegexpPattern, newPattern); err != nil {
					u.Skip(w.state.Settings().StatusMessage().Set("Failed updating a mapping"))
					// TODO: send the error to the notifications
					return
				}
				w.table.Refresh()
				w.appCtx.SettingsViewModel().SaveEditorMappings()
			},
		)
		dial.Show()
	}
}

func (w *EditorMappingsTable) makeOnMappingDelete(selector *editor.Selector, mapping editor.Mapping) func() {
	return func() {
		dialog.ShowConfirm("Are you sure?", "The mapping will be deleted.",
			func(b bool) {
				if !b {
					return
				}
				if err := selector.DeleteMapping(mapping); err != nil {
					u.Skip(w.state.Settings().StatusMessage().Set("Failed deleting a mapping"))
					// TODO: send the error to the notifications
					return
				}
				w.appCtx.SettingsViewModel().SaveEditorMappings()
			},
			w.appCtx.Window(),
		)
	}
}
