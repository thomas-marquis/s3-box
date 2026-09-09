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
	fileMatchersColEditorType = 0
	fileMatchersColPattern    = 1
	fileMatchersColActions    = 2
)

// EditorSelectorTable is a widget that displays and allows editing of file editor matchers.
type EditorSelectorTable struct {
	widget.BaseWidget
	state  *state.State
	appCtx appcontext.AppContext
	table  *widget.Table

	noMatchersBinding binding.String
}

func NewEditorSelectorTable(appCtx appcontext.AppContext) *EditorSelectorTable {
	w := &EditorSelectorTable{
		appCtx:            appCtx,
		state:             appCtx.State(),
		noMatchersBinding: binding.NewString(),
	}

	w.ExtendBaseWidget(w)

	w.table = widget.NewTableWithHeaders(
		func() (rows, cols int) {
			return w.state.Settings().EditorSelectors().Length(), 3
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Selectable = true

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {})
			editBtn.Importance = widget.LowImportance

			rowActions := container.NewHBox(
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

			editBtn := actions.Objects[0].(*widget.Button)

			selectors := w.state.Settings().EditorSelectors()
			if id.Row >= selectors.Length() {
				return
			}

			di, err := selectors.GetItem(id.Row)
			if err != nil {
				return
			}
			bi := di.(binding.Item[*editor.Selector])
			selector, getErr := bi.Get()
			if getErr != nil {
				return
			}

			switch id.Col {
			case fileMatchersColEditorType:
				label.Show()
				actions.Hide()
				label.SetText(selector.Name)

			case fileMatchersColPattern:
				label.Show()
				actions.Hide()
				label.SetText(selector.Pattern)

			case fileMatchersColActions:
				label.SetText("")
				label.Hide()
				actions.Show()

				editBtn.OnTapped = w.makeOnMatcherEdit(selector)
			}
		},
	)

	w.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	w.table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		switch id.Col {
		case fileMatchersColEditorType:
			template.(*widget.Label).SetText("Editor")
		case fileMatchersColPattern:
			template.(*widget.Label).SetText("Pattern")
		}
	}
	w.table.ShowHeaderColumn = false

	w.state.Settings().EditorSelectors().AddListener(binding.NewDataListener(w.table.Refresh))

	return w
}

func (w *EditorSelectorTable) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	w.table.SetColumnWidth(0, 120)
	w.table.SetColumnWidth(1, 300)
	w.table.HideSeparators = true

	c := container.NewBorder(
		widget.NewLabel("Configure the default file editors."),
		nil, nil, nil,
		w.table,
	)

	return widget.NewSimpleRenderer(c)
}

func (w *EditorSelectorTable) Refresh() {
	w.table.Refresh()
}

func (w *EditorSelectorTable) makeEditFormDialog(title string, patternData binding.String, cb func(bool)) *dialog.FormDialog {
	patternEntry := widget.NewEntryWithData(patternData)

	d := dialog.NewForm(title, "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Pattern (regexp)", patternEntry),
	}, cb, w.appCtx.Window())
	d.Resize(fyne.NewSize(450, 120))

	return d
}

func (w *EditorSelectorTable) makeOnMatcherEdit(selector *editor.Selector) func() {
	return func() {
		pattern := binding.NewString()
		u.Skip(pattern.Set(selector.Pattern))

		dial := w.makeEditFormDialog(
			fmt.Sprintf("Update pattern for %s editor", selector.Name),
			pattern,
			func(confirmed bool) {
				if !confirmed {
					return
				}

				p, _ := pattern.Get()
				selector.Pattern = p
				w.table.Refresh()
				w.appCtx.SettingsViewModel().SaveEditorSelectors()
			},
		)
		dial.Show()
	}
}
