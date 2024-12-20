package explorerview

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/explorer"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

const (
	maxFileNameLength = 80
)

type fileDetials struct {
	c *fyne.Container

	downloadBtn       *widget.Button
	previewBtn        *widget.Button
	infoContainer     *fyne.Container
	sizeLabel         *widget.Label
	pathLabel         *widget.Label
	fileIcon          *widget.FileIcon
	lastModifiedLabel *widget.Label
	deleteBtn         *widget.Button
}

func newFileDetails() *fileDetials {
	sizeText := widget.NewLabel("Size")
	sizeText.TextStyle = fyne.TextStyle{Bold: true}
	sizeText.Alignment = fyne.TextAlignTrailing

	lastModifiedText := widget.NewLabel("Last modified")
	lastModifiedText.TextStyle = fyne.TextStyle{Bold: true}
	lastModifiedText.Alignment = fyne.TextAlignTrailing

	sizeLabel := widget.NewLabel("")

	fileIcon := widget.NewFileIcon(nil)
	filepathLabel := widget.NewLabel("")

	lastModifiedLabel := widget.NewLabel("")

	infoContainer := container.NewVBox(
		container.NewHBox(fileIcon, filepathLabel),
		container.NewGridWithColumns(2,
			container.NewVBox(
				sizeText,
				lastModifiedText,
			),
			container.NewVBox(
				sizeLabel,
				lastModifiedLabel,
			),
		),
	)

	downloadBtn := widget.NewButton("Download", func() {})
	deleteBtn := widget.NewButton("Delete", func() {})
	buttonsContainer := container.NewHBox(
		downloadBtn, deleteBtn,
	)
	topContainer := container.NewBorder(
		infoContainer, buttonsContainer,
		nil, nil,
	)
	previewBtn := widget.NewButton("Preview", func() {})

	c := container.NewBorder(
		topContainer, container.NewCenter(previewBtn),
		nil, nil,
	)
	return &fileDetials{
		c:                 c,
		downloadBtn:       downloadBtn,
		previewBtn:        previewBtn,
		infoContainer:     infoContainer,
		sizeLabel:         sizeLabel,
		pathLabel:         filepathLabel,
		fileIcon:          fileIcon,
		lastModifiedLabel: lastModifiedLabel,
		deleteBtn:         deleteBtn,
	}
}

func (d *fileDetials) Object() fyne.CanvasObject {
	return d.c
}

func (f *fileDetials) Update(ctx appcontext.AppContext, file *explorer.RemoteFile) {
	f.sizeLabel.SetText(utils.FormatSizeBytes(file.SizeBytes()))

	var path string
	originalPath := file.Path()
	if len(originalPath) > maxFileNameLength {
		path = ".../" + file.Name()
	} else {
		path = originalPath
	}
	f.pathLabel.SetText(path)

	fileURI := storage.NewFileURI(file.Path())
	f.fileIcon.SetURI(fileURI)

	f.lastModifiedLabel.SetText(file.LastModified().Format("2006-01-02 15:04:05"))

	if file.SizeBytes() <= ctx.ExplorerVM().GetMaxFileSizePreview() {
		f.previewBtn.Show()
		f.previewBtn.OnTapped = func() {
			showFilePreviewDialog(ctx, file)
		}
	} else {
		f.previewBtn.Hide()
	}

	f.downloadBtn.OnTapped = func() {
		saveDialog := dialog.NewFileSave(makeHandleOnDownloadTapped(ctx, file), ctx.W())
		saveDialog.SetFileName(file.Name())
		saveDialog.SetLocation(ctx.ExplorerVM().GetLastSaveDir())
		saveDialog.Show()
	}

	f.deleteBtn.OnTapped = func() {
		dialog.ShowConfirm(
			"Delete file",
			fmt.Sprintf("Are you sure you want to delete %s?", file.Name()),
			makeHandleOnDeleteTapped(ctx, file),
			ctx.W())
	}
}

func makeHandleOnDownloadTapped(ctx appcontext.AppContext, file *explorer.RemoteFile) func(fyne.URIWriteCloser, error) {
	return func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			ctx.Log().Error("Error getting file writer", zap.Error(err))
			// TODO handle error here
			return
		}
		if writer == nil {
			return
		}
		localDestFilePath := writer.URI().Path()
		if err := ctx.ExplorerVM().DownloadFile(file, localDestFilePath); err != nil {
			ctx.Log().Error("Error downloading file", zap.Error(err))
			// TODO handle error here
		}
		if err := ctx.ExplorerVM().SetLastSaveDir(localDestFilePath); err != nil {
			ctx.Log().Error("Error setting last save dir", zap.Error(err))
		}
		dialog.ShowInformation("Download", "File downloaded", ctx.W())
	}
}

func makeHandleOnDeleteTapped(ctx appcontext.AppContext, file *explorer.RemoteFile) func(bool) {
	return func(confirmed bool) {
		if !confirmed {
			return
		}
		if err := ctx.ExplorerVM().DeleteFile(file); err != nil {
			ctx.Log().Error("Error deleting file", zap.Error(err))
			// TODO handle error here
		}

		// TODO: not working, need to reimplement refresh
		// if err := ctx.Vm().RefreshDir(file.ParentDir()); err != nil {
		// 	dialog.ShowError(err, ctx.W()) // TODO better error handling
		// 	return
		// }
		dialog.ShowInformation("Delete", "File deleted", ctx.W())
	}
}
