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
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
)

const (
	maxFileNameLength = 60
)

type FileDetails struct {
	widget.BaseWidget

	appCtx appcontext.AppContext

	pathLabel *widget.Label
	fileIcon  *widget.FileIcon
	tags      *TagsTable

	downloadAction *ToolbarButton
	deleteAction   *ToolbarButton
	openAction     *ToolbarButton
	renameAction   *ToolbarButton
	openWithAction *ToolbarButton

	fileSizeBinding     binding.String
	lastModifiedBinding binding.String
	maxFileSizeListener binding.DataListener

	currentSelectedFile *directory.File
}

func NewFileDetails(appCtx appcontext.AppContext) *FileDetails {
	fileIcon := widget.NewFileIcon(nil)
	filepathLabel := widget.NewLabel("")
	filepathLabel.Selectable = true

	w := &FileDetails{
		appCtx:    appCtx,
		pathLabel: filepathLabel,
		fileIcon:  fileIcon,

		fileSizeBinding:     binding.NewString(),
		lastModifiedBinding: binding.NewString(),
		maxFileSizeListener: binding.NewDataListener(func() {}),

		downloadAction: NewToolbarButton("Download", theme.DownloadIcon(), func() {}),
		deleteAction:   NewToolbarButton("Delete", theme.DeleteIcon(), func() {}),
		openAction:     NewToolbarButton("Open", theme.DocumentCreateIcon(), func() {}),
		renameAction:   NewToolbarButton("Rename", theme.FileTextIcon(), func() {}),
		openWithAction: NewToolbarButton("Open with...", theme.DocumentCreateIcon(), func() {}),

		currentSelectedFile: nil,
	}

	w.tags = NewTagsTable(w.appCtx, w.appCtx.TagsViewModel())

	w.ExtendBaseWidget(w)
	return w
}

func (w *FileDetails) CreateRenderer() fyne.WidgetRenderer {
	fileSize := widget.NewLabelWithData(w.fileSizeBinding)
	fileSize.Selectable = true

	copyPath := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if w.currentSelectedFile == nil {
			return
		}
		fyne.CurrentApp().Clipboard().SetContent(w.currentSelectedFile.FullPath())
	})

	lastModified := widget.NewLabelWithData(w.lastModifiedBinding)
	lastModified.Selectable = true

	fileInfosTable := container.NewGridWithColumns(2,
		widget.NewLabelWithStyle("Size", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
		fileSize,
		widget.NewLabelWithStyle("Last modified", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
		lastModified,
	)

	tagsGroup := widget.NewAccordion(
		widget.NewAccordionItem("Tags", w.tags),
	)
	tagsGroup.OpenAll()

	return widget.NewSimpleRenderer(
		container.NewBorder(
			container.NewVBox(
				container.NewBorder(nil, nil,
					container.NewHBox(w.fileIcon, w.pathLabel),
					copyPath),
				makeSeparator(),
			),
			nil, nil, nil,
			container.NewBorder(
				container.NewVBox(
					container.New(
						layout.NewCustomPaddedLayout(0, 0, 5, 5),
						widget.NewToolbar(
							w.openAction,
							w.openWithAction,
						),
					),
					container.New(
						layout.NewCustomPaddedLayout(0, 0, 5, 5),
						widget.NewToolbar(
							w.downloadAction,
							w.renameAction,
							w.deleteAction,
						),
					),
					container.New(
						layout.NewCustomPaddedLayout(30, 0, 5, 5),
						fileInfosTable,
					),
					makeSeparator()),
				nil, nil, nil,
				container.NewPadded(tagsGroup),
			),
		),
	)
}

func (w *FileDetails) Select(file *directory.File) {
	exVm := w.appCtx.ExplorerViewModel()
	edVm := w.appCtx.EditorViewModel()
	st := w.appCtx.State()

	w.currentSelectedFile = file

	w.tags.Select(file.TagSet())

	var path string
	originalPath := file.FullPath()
	if len(originalPath) > maxFileNameLength {
		path = file.Name().String()
		if len(path) > maxFileNameLength {
			path = "..." + path[len(path)-maxFileNameLength+3:]
		}
		path = ".../" + path
	} else {
		path = originalPath
	}
	w.pathLabel.SetText(path)

	fileURI := storage.NewFileURI(file.FullPath())
	w.fileIcon.SetURI(fileURI)

	u.Skip(w.lastModifiedBinding.Set(file.LastModified().Format("2006-01-02 15:04:05")))
	u.Skip(w.fileSizeBinding.Set(humanize.Bytes(file.SizeBytes())))

	s := w.appCtx.State().Editors().Selector()

	st.Settings().EditorFileSizeLimitBytes().RemoveListener(w.maxFileSizeListener)
	dl := binding.NewDataListener(func() {
		fileIsEditable := st.Explorer().IsFileEditable(file)
		defaultFactory := st.Editors().GetDefaultFactory(file.FullPath())
		defaultIsNonEditable := defaultFactory != nil && !defaultFactory.Capabilities().Editable

		if fileIsEditable || defaultIsNonEditable {
			w.openAction.Enable()
		} else {
			w.openAction.Disable()
		}
	})
	st.Settings().EditorFileSizeLimitBytes().AddListener(dl)
	w.maxFileSizeListener = dl

	w.openAction.SetOnTapped(func() {
		ed, err := edVm.Open(file)
		if err != nil && !errors.Is(err, viewmodel.ErrEditorAlreadyOpened) {
			dialog.ShowError(err, w.appCtx.Window())
		}
		showEditor(ed)
	})

	registered := s.RegisteredEditors()

	showOnlyNonEditable := !st.Explorer().IsFileEditable(file)

	var filteredEditors []editor.Factory
	if showOnlyNonEditable {
		filteredEditors = s.FilterRegistered(editor.Capabilities{Editable: false})
	} else {
		filteredEditors = registered
	}

	openWithItems := make([]*fyne.MenuItem, len(filteredEditors))
	for i, f := range filteredEditors {
		openWithItems[i] = fyne.NewMenuItem(f.DisplayLabel(), func() {
			ed, err := edVm.OpenWith(file, f)
			if err != nil {
				dialog.ShowError(err, w.appCtx.Window())
			}
			showEditor(ed)
		})
	}

	w.openWithAction.SetOnTapped(func() {
		widget.NewPopUpMenu(
			fyne.NewMenu("", openWithItems...),
			w.appCtx.Window().Canvas()).
			ShowAtRelativePosition(fyne.NewPos(170, 20), w.openAction.ToolbarObject())
	})

	w.downloadAction.SetOnTapped(func() {
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(fmt.Errorf("error saving file: %w", err), w.appCtx.Window())
				return
			}
			if writer == nil {
				return
			}
			localDestFilePath := writer.URI().Path()
			exVm.DownloadFile(file, localDestFilePath)
			fyne.CurrentApp().SendNotification(fyne.NewNotification("File download", "success"))
		}, w.appCtx.Window())
		saveDialog.SetFileName(file.Name().String())
		saveDialog.SetLocation(
			u.SkipV(st.Explorer().DownloadLocation().Get()))
		saveDialog.Show()
	})

	w.renameAction.SetOnTapped(func() {
		var d *dialog.FormDialog
		nameEntry := NewEntryWithShortcuts([]ActionShortcuts{
			{
				Shortcuts: []desktop.CustomShortcut{
					{KeyName: fyne.KeyReturn, Modifier: fyne.KeyModifierControl},
				},
				Callback: func() { d.Submit() },
			},
			{
				Shortcuts: []desktop.CustomShortcut{
					{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl},
				},
				Callback: func() { d.Dismiss() },
			},
		})
		nameEntry.SetText(file.Name().String())
		d = dialog.NewForm(
			"Rename file",
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
				exVm.RenameFile(file, newName)
			},
			w.appCtx.Window(),
		)
		d.Resize(fyne.NewSize(400, 150))
		d.Show()
	})

	w.deleteAction.SetOnTapped(func() {
		dialog.ShowConfirm("Delete file",
			fmt.Sprintf("Are you sure you want to delete '%s'?", file.Name()),
			func(b bool) {
				if b {
					exVm.DeleteFile(file)
				}
			}, w.appCtx.Window())
	})
	if st.Connection().IsReadOnly() {
		w.deleteAction.Disable()
		w.renameAction.Disable()
	}
}

func makeSeparator() fyne.CanvasObject {
	return container.New(
		layout.NewCustomPaddedLayout(10, 20, 0, 0),
		widget.NewSeparator(),
	)
}

func showEditor(ed editor.Editor) {
	ed.Window().SetContent(ed.CreateWidget())
	ed.Window().SetFixedSize(false)
	ed.Window().Resize(fyne.NewSize(700, 500))
	ed.Window().Show()
	ed.Window().RequestFocus()
}
