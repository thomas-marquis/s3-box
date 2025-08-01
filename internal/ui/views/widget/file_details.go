package widget

import (
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
	"github.com/thomas-marquis/s3-box/internal/utils"
)

const (
	maxFileNameLength = 80
)

type FileDetails struct {
	widget.BaseWidget

	appCtx appcontext.AppContext

	pathLabel *widget.Label
	fileIcon  *widget.FileIcon

	downloadAction *ToolbarButton
	previewAction  *ToolbarButton
	deleteAction   *ToolbarButton

	fileSizeBinding     binding.String
	lastModifiedBinding binding.String
}

func NewFileDetails(appCtx appcontext.AppContext) *FileDetails {
	fileIcon := widget.NewFileIcon(nil)
	filepathLabel := widget.NewLabel("")

	w := &FileDetails{
		appCtx:    appCtx,
		pathLabel: filepathLabel,
		fileIcon:  fileIcon,

		fileSizeBinding:     binding.NewString(),
		lastModifiedBinding: binding.NewString(),

		downloadAction: NewToolbarButton("Download", theme.DownloadIcon(), func() {}),
		previewAction:  NewToolbarButton("Preview", theme.VisibilityIcon(), func() {}),
		deleteAction:   NewToolbarButton("Delete", theme.DeleteIcon(), func() {}),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *FileDetails) CreateRenderer() fyne.WidgetRenderer {
	fileInfosTable := container.NewGridWithColumns(2,
		widget.NewLabelWithStyle("Size", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithData(w.fileSizeBinding),

		widget.NewLabelWithStyle("Last modified", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithData(w.lastModifiedBinding),
	)

	actionToolbar := widget.NewToolbar(
		w.downloadAction,
		w.previewAction,
		w.deleteAction,
	)

	return widget.NewSimpleRenderer(
		container.NewVBox(
			container.NewHBox(w.fileIcon, w.pathLabel),
			container.New(
				layout.NewCustomPaddedLayout(10, 20, 0, 0),
				widget.NewSeparator(),
			),
			container.New(
				layout.NewCustomPaddedLayout(0, 0, 5, 5),
				actionToolbar,
			),
			container.New(
				layout.NewCustomPaddedLayout(30, 0, 5, 5),
				fileInfosTable,
			),
		),
	)
}

func (w *FileDetails) Render(file *directory.File) {
	vm := w.appCtx.ExplorerViewModel()

	var path string
	originalPath := file.FullPath()
	if len(originalPath) > maxFileNameLength {
		path = ".../" + file.Name().String()
	} else {
		path = originalPath
	}
	w.pathLabel.SetText(path)

	fileURI := storage.NewFileURI(file.FullPath())
	w.fileIcon.SetURI(fileURI)

	w.lastModifiedBinding.Set(file.LastModified().Format("2006-01-02 15:04:05"))
	w.fileSizeBinding.Set(utils.FormatSizeBytes(file.SizeBytes()))

	if file.SizeBytes() <= w.appCtx.SettingsViewModel().CurrentMaxFilePreviewSizeBytes() {
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
			if err := vm.DownloadFile(file, localDestFilePath); err != nil {
				outErr := fmt.Errorf("error downloading file: %w", err)
				dialog.ShowError(outErr, w.appCtx.Window())
				return
			}
			vm.UpdateLastDownloadLocation(localDestFilePath)
			dialog.ShowInformation("Download", "File downloaded", w.appCtx.Window())
		}, w.appCtx.Window())
		saveDialog.SetFileName(file.Name().String())
		saveDialog.SetLocation(vm.LastDownloadLocation())
		saveDialog.Show()
	})

	w.deleteAction.SetOnTapped(func() {
		dialog.ShowConfirm("Delete file",
			fmt.Sprintf("Are you sure you want to delete '%s'?", file.Name()),
			func(b bool) {
				if b {
					if err := vm.DeleteFile(file); err != nil {
						dialog.ShowError(err, w.appCtx.Window())
					} else {
						dialog.ShowInformation("Delete", "File deleted", w.appCtx.Window())
					}
				}
			}, w.appCtx.Window())
	})
	if w.appCtx.ConnectionViewModel().IsReadOnly() {
		w.deleteAction.Disable()
	}
}
