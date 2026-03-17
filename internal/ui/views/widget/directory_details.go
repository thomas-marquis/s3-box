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

	pathLabel        *widget.Label
	renameErrContent *renameFailedPanel

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
		toolbar:            toolbar,
		newDirectoryAction: createDirAction,
		uploadAction:       uploadAction,
		createFileAction:   createFileAction,
		renameAction:       renameAction,
		loadingBar:         loadingBar,
		renameErrContent:   newRenameFailedPanel(appCtx.Window()),
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

	content := container.NewVBox(
		w.renameErrContent,
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

	w.uploadAction.SetOnTapped(w.makeOnUpload(vm, dir))
	w.newDirectoryAction.SetOnTapped(w.makeOnCreateDirectory(vm, dir))
	w.createFileAction.SetOnTapped(w.makeOnCreateFile(vm, dir))
	w.renameAction.SetOnTapped(w.makeOnRename(vm, dir))

	if dir.IsRoot() {
		w.renameAction.Disable()
	} else {
		w.renameAction.Enable()
	}

	if dir.HasError() {
		w.renameErrContent.Show()
		w.renameErrContent.SetMessage(dir.Status().Title() + ": " + dir.Status().Message())
		w.renameErrContent.SetCallbacks(func() {
			if err := vm.ResumeRename(dir); err != nil {
				dialog.ShowError(err, w.appCtx.Window())
			}
		}, func() {
			if err := vm.RollbackRename(dir); err != nil {
				dialog.ShowError(err, w.appCtx.Window())
			}
		}, func() {
			if err := vm.AbortRename(dir); err != nil {
				dialog.ShowError(err, w.appCtx.Window())
			}
		})
	} else {
		w.renameErrContent.Hide()
		w.renameErrContent.OnResume = func() {}
		w.renameErrContent.OnRollback = func() {}
		w.renameErrContent.OnAbort = func() {}
	}

	if w.appCtx.ConnectionViewModel().IsReadOnly() || !dir.IsLoaded() || dir.HasError() {
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

type renameFailedPanel struct {
	widget.BaseWidget

	window      fyne.Window
	statusLabel *OpenableLabel

	OnResume   func()
	OnRollback func()
	OnAbort    func()

	resumeBtn   *widget.Button
	rollbackBtn *widget.Button
	abortBtn    *widget.Button
}

func newRenameFailedPanel(window fyne.Window) *renameFailedPanel {
	w := &renameFailedPanel{
		window:     window,
		OnResume:   func() {},
		OnRollback: func() {},
		OnAbort:    func() {},
	}
	w.ExtendBaseWidget(w)
	return w
}

const renameExplanation = `
### Ooops... it appears renaming failed...


Renaming a directory is a complex process on S3. 

Something probably went wrong, leaving this directory (and the new named one) 

in an inconsistent state (e.g. some of your files have been renamed, but some others not).


But don't worry (too much), here are your options:


* 1. Resume the renaming operation (recommended)
* 2. Rollback to the old name
* 3. Abort the process completely and leave everything as is.
`

func (w *renameFailedPanel) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	statusLabel := NewOpenableLabel("", w.window)
	statusLabel.Selectable = false
	statusLabel.Alignment = fyne.TextAlignLeading
	statusLabel.Truncation = fyne.TextTruncateEllipsis
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	w.resumeBtn = widget.NewButton("Resume", func() {})
	w.rollbackBtn = widget.NewButton("Rollback", func() {})
	w.abortBtn = widget.NewButton("Abort", func() {})

	staticLabel := widget.NewRichTextFromMarkdown(renameExplanation)
	staticLabel.Scroll = fyne.ScrollHorizontalOnly

	c := container.NewVBox(
		statusLabel,
		staticLabel,
		container.NewHBox(
			w.resumeBtn,
			w.rollbackBtn,
			w.abortBtn,
		),
	)

	w.statusLabel = statusLabel

	return widget.NewSimpleRenderer(c)
}

func (w *renameFailedPanel) SetMessage(msg string) {
	w.statusLabel.SetText(msg)
}

func (w *renameFailedPanel) SetCallbacks(resume, rollback, abort func()) {
	w.resumeBtn.OnTapped = resume
	w.rollbackBtn.OnTapped = rollback
	w.abortBtn.OnTapped = abort
}
