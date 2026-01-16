package widget

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
)

type DirectoryDetails struct {
	widget.BaseWidget

	appCtx appcontext.AppContext

	pathLabel *widget.Label

	toolbar            *widget.Toolbar
	newDirectoryAction *ToolbarButton
	uploadAction       *ToolbarButton
}

func NewDirectoryDetails(appCtx appcontext.AppContext) *DirectoryDetails {
	pathLabel := widget.NewLabel("")

	createDirAction := NewToolbarButton("New directory", theme.FolderNewIcon(), func() {})
	uploadAction := NewToolbarButton("Upload file", theme.UploadIcon(), func() {})
	toolbar := widget.NewToolbar(
		createDirAction,
		uploadAction,
	)

	w := &DirectoryDetails{
		appCtx:             appCtx,
		pathLabel:          pathLabel,
		toolbar:            toolbar,
		newDirectoryAction: createDirAction,
		uploadAction:       uploadAction,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *DirectoryDetails) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewVBox(
		container.NewHBox(
			widget.NewIcon(theme.FolderIcon()),
			w.pathLabel,
		),
		container.New(
			layout.NewCustomPaddedLayout(10, 20, 0, 0),
			widget.NewSeparator(),
		),
		container.New(
			layout.NewCustomPaddedLayout(0, 0, 5, 5),
			w.toolbar,
		),
	))
}

func (w *DirectoryDetails) Render(dir *directory.Directory) {
	vm := w.appCtx.ExplorerViewModel()

	w.pathLabel.SetText(dir.Path().String())

	w.uploadAction.SetOnTapped(w.makeOnUpload(vm, dir))
	w.newDirectoryAction.SetOnTapped(w.makeOnCreateDirectory(vm, dir))

	if w.appCtx.ConnectionViewModel().IsReadOnly() {
		w.newDirectoryAction.Disable()
		w.uploadAction.Disable()
	}
}

func (w *DirectoryDetails) makeOnUpload(vm viewmodel.ExplorerViewModel, dir *directory.Directory) func() {
	return func() {
		selectDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w.appCtx.Window())
				return
			}
			if reader == nil {
				return
			}

			localDestFilePath := reader.URI().Path()
			vm.UploadFile(localDestFilePath, dir)
			vm.UpdateLastUploadLocation(localDestFilePath)
			dialog.ShowInformation("Upload", "AttachedFile uploaded", w.appCtx.Window())
		}, w.appCtx.Window())

		selectDialog.SetLocation(vm.LastUploadLocation())
		selectDialog.Show()
	}
}

func (w *DirectoryDetails) makeOnCreateDirectory(vm viewmodel.ExplorerViewModel, dir *directory.Directory) func() {
	return func() {
		nameEntry := widget.NewEntry()
		dialog.ShowForm(
			fmt.Sprintf("New directory under %s", dir.Name()),
			"Create",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Type", nameEntry),
			},
			func(ok bool) {
				if !ok {
					return
				}
				name := nameEntry.Text
				vm.CreateEmptyDirectory(dir, name)
			},
			w.appCtx.Window(),
		)
	}
}
