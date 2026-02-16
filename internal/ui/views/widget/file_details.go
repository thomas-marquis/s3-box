package widget

import (
	"context"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/utils"
)

const (
	maxFileNameLength = 60
)

type FileDetails struct {
	widget.BaseWidget

	appCtx appcontext.AppContext

	pathLabel *widget.Label
	fileIcon  *widget.FileIcon

	downloadAction *ToolbarButton
	previewAction  *ToolbarButton
	deleteAction   *ToolbarButton
	editAction     *ToolbarButton

	actionToolbar *widget.Toolbar

	fileSizeBinding     binding.String
	lastModifiedBinding binding.String

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

		downloadAction: NewToolbarButton("Download", theme.DownloadIcon(), func() {}),
		previewAction:  NewToolbarButton("Preview", theme.VisibilityIcon(), func() {}),
		deleteAction:   NewToolbarButton("Delete", theme.DeleteIcon(), func() {}),
		editAction:     NewToolbarButton("Edit", theme.DocumentCreateIcon(), func() {}),

		currentSelectedFile: nil,
	}

	w.actionToolbar = widget.NewToolbar(
		w.downloadAction,
		w.previewAction,
		w.editAction,
		w.deleteAction,
	)

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

	return widget.NewSimpleRenderer(
		container.NewVBox(
			container.NewBorder(nil, nil,
				container.NewHBox(w.fileIcon, w.pathLabel),
				copyPath),
			container.New(
				layout.NewCustomPaddedLayout(10, 20, 0, 0),
				widget.NewSeparator(),
			),
			container.New(
				layout.NewCustomPaddedLayout(0, 0, 5, 5),
				w.actionToolbar,
			),
			container.New(
				layout.NewCustomPaddedLayout(30, 0, 5, 5),
				fileInfosTable,
			),
		),
	)
}

func (w *FileDetails) Select(file *directory.File) {
	exVm := w.appCtx.ExplorerViewModel()
	edVm := w.appCtx.EditorVewModel()

	w.currentSelectedFile = file

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

	w.lastModifiedBinding.Set(file.LastModified().Format("2006-01-02 15:04:05")) //nolint:errcheck
	w.fileSizeBinding.Set(utils.FormatSizeBytes(file.SizeBytes()))               //nolint:errcheck

	if file.SizeBytes() <= w.appCtx.SettingsViewModel().CurrentFileSizeLimitBytes() {
		w.previewAction.Enable()
		w.previewAction.SetOnTapped(func() {
			viewerDialog := dialog.NewCustom(
				file.Name().String(),
				"Close",
				NewFileViewer(w.appCtx, file),
				w.appCtx.Window(),
			)
			viewerDialog.Resize(fyne.NewSize(700, 500))
			viewerDialog.Show()
		})
	} else {
		w.previewAction.Disable()
	}

	w.editAction.SetOnTapped(func() {
		ctx, cancel := context.WithCancel(context.Background())

		oe, err := edVm.Open(ctx, file)
		if err != nil {
			if !errors.Is(err, viewmodel.ErrEditorAlreadyOpened) {
				dialog.ShowError(err, w.appCtx.Window())
			}
			cancel()
			return
		}
		editor := NewFileEditor(oe)

		oe.Window.SetOnClosed(func() {
			cancel()
			edVm.Close(oe)
			w.editAction.Icon = theme.DocumentCreateIcon()
			w.actionToolbar.Refresh()
		})
		oe.Window.SetContent(editor)
		oe.Window.SetFixedSize(false)
		oe.Window.Resize(fyne.NewSize(700, 500))
		oe.Window.Show()

		oe.Window.RequestFocus()
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
			if err := exVm.UpdateLastDownloadLocation(localDestFilePath); err != nil { //nolint:staticcheck
				// TODO: handle error
			}
			fyne.CurrentApp().SendNotification(fyne.NewNotification("File download", "success"))
		}, w.appCtx.Window())
		saveDialog.SetFileName(file.Name().String())
		saveDialog.SetLocation(exVm.LastDownloadLocation())
		saveDialog.Show()
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
	if w.appCtx.ConnectionViewModel().IsReadOnly() {
		w.deleteAction.Disable()
		w.editAction.Disable()
	}
}
