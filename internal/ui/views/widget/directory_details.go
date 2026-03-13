package widget

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
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

	pathLabel         *widget.Label
	statusLabel       *OpenableLabel
	actionRequiredBtn *widget.Button

	toolbar            *widget.Toolbar
	newDirectoryAction *ToolbarButton
	uploadAction       *ToolbarButton
	createFileAction   *ToolbarButton
	renameAction       *ToolbarButton
	loadingBar         *widget.ProgressBarInfinite
}

func NewDirectoryDetails(appCtx appcontext.AppContext) *DirectoryDetails {
	pathLabel := widget.NewLabel("")
	pathLabel.Selectable = true

	statusLabel := NewOpenableLabel("", appCtx.Window())
	statusLabel.Selectable = false
	statusLabel.Alignment = fyne.TextAlignLeading
	statusLabel.Truncation = fyne.TextTruncateEllipsis
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	statusLabel.Hide()

	actionRequiredBtn := widget.NewButton("Action required", func() {})
	actionRequiredBtn.Hide()

	createDirAction := NewToolbarButton("New empty directory", theme.FolderNewIcon(), func() {})
	createFileAction := NewToolbarButton("New empty file", theme.ContentAddIcon(), func() {})
	uploadAction := NewToolbarButton("Upload file", theme.UploadIcon(), func() {})
	renameAction := NewToolbarButton("Rename", theme.FileTextIcon(), func() {})
	toolbar := widget.NewToolbar(
		createDirAction,
		createFileAction,
		uploadAction,
		renameAction,
	)
	loadingBar := widget.NewProgressBarInfinite()
	loadingBar.Hide()

	w := &DirectoryDetails{
		appCtx:             appCtx,
		pathLabel:          pathLabel,
		statusLabel:        statusLabel,
		actionRequiredBtn:  actionRequiredBtn,
		toolbar:            toolbar,
		newDirectoryAction: createDirAction,
		uploadAction:       uploadAction,
		createFileAction:   createFileAction,
		renameAction:       renameAction,
		loadingBar:         loadingBar,
	}
	w.ExtendBaseWidget(w)

	appCtx.ExplorerViewModel().IsSelectedDirectoryLoading().AddListener(binding.NewDataListener(func() {
		loading, _ := appCtx.ExplorerViewModel().IsSelectedDirectoryLoading().Get()
		if loading {
			loadingBar.Show()
			loadingBar.Start()
		} else {
			w.loadingBar.Stop()
			w.loadingBar.Hide()
		}
	}))

	return w
}

func (w *DirectoryDetails) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	copyPath := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		sd := w.appCtx.ExplorerViewModel().SelectedDirectory()
		if sd == nil {
			return
		}
		fyne.CurrentApp().Clipboard().SetContent(sd.Path().String())
	})

	arContainer := container.NewCenter(
		w.actionRequiredBtn,
	)

	content := container.NewStack(
		w.statusLabel,
		arContainer,
	)

	return widget.NewSimpleRenderer(container.NewVBox(
		w.loadingBar,
		container.NewBorder(nil, nil,
			container.NewHBox(
				widget.NewIcon(theme.FolderIcon()),
				w.pathLabel,
			),
			copyPath),
		container.New(
			layout.NewCustomPaddedLayout(10, 20, 0, 0),
			widget.NewSeparator(),
		),
		container.New(
			layout.NewCustomPaddedLayout(0, 0, 5, 5),
			w.toolbar,
		),
		container.New(
			layout.NewCustomPaddedLayout(10, 20, 0, 0),
			widget.NewSeparator(),
		),
		content,
	))
}

func (w *DirectoryDetails) Select(dir *directory.Directory) {
	vm := w.appCtx.ExplorerViewModel()

	if dir.IsLoading() {
		w.loadingBar.Show()
		w.loadingBar.Start()
	} else {
		w.loadingBar.Stop()
		w.loadingBar.Hide()
	}

	var path string
	originalPath := dir.Path().String()
	if len(originalPath) > maxFileNameLength {
		path = dir.Name()
		if len(path) > maxFileNameLength {
			path = "..." + path[len(path)-maxFileNameLength+3:]
		}
		path = ".../" + path
	} else {
		path = originalPath
	}
	w.pathLabel.SetText(path)

	if dir.Status() != nil {
		w.statusLabel.SetText(dir.Status().Title() + ": " + dir.Status().Message())
		w.statusLabel.Show()
	} else {
		w.statusLabel.SetText("")
		w.statusLabel.Hide()
	}

	w.uploadAction.SetOnTapped(w.makeOnUpload(vm, dir))
	w.newDirectoryAction.SetOnTapped(w.makeOnCreateDirectory(vm, dir))
	w.createFileAction.SetOnTapped(w.makeOnCreateFile(vm, dir))
	w.renameAction.SetOnTapped(w.makeOnRename(vm, dir))

	if dir.IsRoot() {
		w.renameAction.Disable()
	} else {
		w.renameAction.Enable()
	}

	if dir.IsResumable() {
		w.actionRequiredBtn.Show()
		w.actionRequiredBtn.OnTapped = w.makeOnResumeRename(vm, dir)
	} else {
		w.actionRequiredBtn.Hide()
		w.actionRequiredBtn.OnTapped = func() {}
	}

	if w.appCtx.ConnectionViewModel().IsReadOnly() || !dir.IsLoaded() || dir.IsResumable() {
		w.newDirectoryAction.Disable()
		w.uploadAction.Disable()
		w.createFileAction.Disable()
		w.renameAction.Disable()
	} else {
		w.newDirectoryAction.Enable()
		w.uploadAction.Enable()
		w.createFileAction.Enable()
	}
}

func (w *DirectoryDetails) makeOnResumeRename(vm viewmodel.ExplorerViewModel, dir *directory.Directory) func() {
	return func() {
		vm.ResumeRename(dir)
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
			if err := vm.UploadFile(localDestFilePath, dir, false); err != nil {
				if errors.Is(err, directory.ErrAlreadyExists) {
					dialog.ShowConfirm(
						"This file already exists",
						"Do you want to overwrite it?",
						func(b bool) {
							if b {
								if err := vm.UploadFile(localDestFilePath, dir, true); err != nil {
									dialog.ShowError(err, w.appCtx.Window())
								}
							}
						},
						w.appCtx.Window())
					return
				}
				dialog.ShowError(err, w.appCtx.Window())
			}
			vm.UpdateLastUploadLocation(localDestFilePath)
		}, w.appCtx.Window())

		selectDialog.SetLocation(vm.LastUploadLocation())
		selectDialog.Show()
	}
}

func (w *DirectoryDetails) makeOnCreateDirectory(vm viewmodel.ExplorerViewModel, dir *directory.Directory) func() {
	return func() {
		var d *dialog.FormDialog
		nameEntry := entryWithShortcuts(func() { d.Submit() }, func() { d.Dismiss() })
		d = dialog.NewForm(
			fmt.Sprintf("New directory under %s", dir.Name()),
			"Create",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Name", nameEntry),
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
		d.Resize(fyne.NewSize(400, 150))
		d.Show()
	}
}

func (w *DirectoryDetails) makeOnCreateFile(vm viewmodel.ExplorerViewModel, dir *directory.Directory) func() {
	return func() {
		var d *dialog.FormDialog
		nameEntry := entryWithShortcuts(func() { d.Submit() }, func() { d.Dismiss() })
		d = dialog.NewForm(
			fmt.Sprintf("New empty file in %s", dir.Path()),
			"Create",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Name", nameEntry),
			},
			func(ok bool) {
				if !ok {
					return
				}
				name := nameEntry.Text
				vm.CreateEmptyFile(dir, name)
			},
			w.appCtx.Window(),
		)
		d.Resize(fyne.NewSize(400, 150))
		d.Show()
	}
}

func (w *DirectoryDetails) makeOnRename(vm viewmodel.ExplorerViewModel, dir *directory.Directory) func() {
	return func() {
		var d *dialog.FormDialog
		nameEntry := entryWithShortcuts(func() { d.Submit() }, func() { d.Dismiss() })
		nameEntry.SetText(dir.Name())
		d = dialog.NewForm(
			"Rename directory",
			"Rename",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("New name", nameEntry),
			},
			func(ok bool) {
				if !ok {
					return
				}
				newName := nameEntry.Text
				vm.RenameDirectory(dir, newName)
			},
			w.appCtx.Window(),
		)
		d.Resize(fyne.NewSize(400, 150))
		d.Show()
	}
}

func entryWithShortcuts(onSubmit, onDismiss func()) *EntryWithShortcuts {
	return NewEntryWithShortcuts([]ActionShortcuts{
		{
			Shortcuts: []desktop.CustomShortcut{
				{KeyName: fyne.KeyReturn, Modifier: fyne.KeyModifierControl},
			},
			Callback: onSubmit,
		},
		{
			Shortcuts: []desktop.CustomShortcut{
				{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl},
			},
			Callback: onDismiss,
		},
	})
}
