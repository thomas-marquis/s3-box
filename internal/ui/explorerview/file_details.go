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
	"fyne.io/fyne/v2/theme"
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
	deleteBtn         *widget.Button
	infoContainer     *fyne.Container
	sizeLabel         *widget.Label
	pathLabel         *widget.Label
	fileIcon          *widget.FileIcon
	lastModifiedLabel *widget.Label
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
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
	buttonsContainer := container.NewHBox(
		downloadBtn,
		deleteBtn,
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
		deleteBtn:         deleteBtn,
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

func (f *FileDetials) Update(ctx appcontext.AppContext, file *explorer.S3File) {
	f.sizeLabel.SetText(utils.FormatSizeBytes(file.SizeBytes))

	var path string
	originalPath := file.ID.String()
	if len(originalPath) > maxFileNameLength {
		path = ".../" + file.Name
	} else {
		path = originalPath
	}
	f.pathLabel.SetText(path)

	fileURI := storage.NewFileURI(file.ID.String())
	f.fileIcon.SetURI(fileURI)

	f.lastModifiedLabel.SetText(file.LastModified.Format("2006-01-02 15:04:05"))

	if file.SizeBytes <= ctx.ExplorerViewModel().GetMaxFileSizePreview() {
		f.previewBtn.Show()
		f.previewBtn.OnTapped = func() {
			showFilePreviewDialog(ctx, file)
		}
	} else {
		f.previewBtn.Hide()
	}

	f.downloadBtn.OnTapped = func() {
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				ctx.L().Error("Error getting file writer", zap.Error(err))
				// TODO handle error here
				return
			}
			if writer == nil {
				return
			}
			localDestFilePath := writer.URI().Path()
			if err := ctx.ExplorerViewModel().DownloadFile(file, localDestFilePath); err != nil {
				ctx.L().Error("Error downloading file", zap.Error(err))
				// TODO handle error here
			}
			if err := ctx.ExplorerViewModel().SetLastSaveDir(localDestFilePath); err != nil {
				ctx.L().Error("Error setting last save dir", zap.Error(err))
			}
			dialog.ShowInformation("Download", "File downloaded", ctx.Window())
		}, ctx.Window())
		saveDialog.SetFileName(file.Name)
		saveDialog.SetLocation(ctx.ExplorerViewModel().GetLastSaveDir())
		saveDialog.Show()
	}

	f.deleteBtn.OnTapped = func() {
		dialog.ShowConfirm("Delete file", fmt.Sprintf("Are you sure you want to delete '%s'?", file.Name), func(b bool) {
			if b {
				if err := ctx.ExplorerViewModel().DeleteFile(file); err != nil {
					ctx.L().Error("Error deleting file", zap.Error(err))
					dialog.ShowError(err, ctx.Window())
				} else {
					dialog.ShowInformation("Delete", "File deleted", ctx.Window())
				}
			}
		}, ctx.Window())
	}
}
