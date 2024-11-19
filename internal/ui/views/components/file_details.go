package components

import (
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

type FileDetials struct {
	c *fyne.Container

	downloadBtn       *widget.Button
	previewBtn        *widget.Button
	infoContainer     *fyne.Container
	sizeLabel         *widget.Label
	pathLabel         *widget.Label
	fileIcon          *widget.FileIcon
	lastModifiedLabel *widget.Label
}

func NewFileDetails() *FileDetials {
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
	buttonsContainer := container.NewHBox(
		downloadBtn,
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
	return &FileDetials{
		c:                 c,
		downloadBtn:       downloadBtn,
		previewBtn:        previewBtn,
		infoContainer:     infoContainer,
		sizeLabel:         sizeLabel,
		pathLabel:         filepathLabel,
		fileIcon:          fileIcon,
		lastModifiedLabel: lastModifiedLabel,
	}
}

func (d *FileDetials) Object() fyne.CanvasObject {
	return d.c
}

func (f *FileDetials) Update(ctx appcontext.AppContext, file *explorer.RemoteFile) {
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

	if file.SizeBytes() <= ctx.Vm().GetMaxFileSizePreview() {
		f.previewBtn.Show()
		f.previewBtn.OnTapped = func() {
			ShowFilePreviewDialog(ctx, file)
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
			if err := ctx.Vm().DownloadFile(file, localDestFilePath); err != nil {
				ctx.L().Error("Error downloading file", zap.Error(err))
				// TODO handle error here
			}
			if err := ctx.Vm().SetLastSaveDir(localDestFilePath); err != nil {
				ctx.L().Error("Error setting last save dir", zap.Error(err))
			}
			dialog.ShowInformation("Download", "File downloaded", ctx.W())
		}, ctx.W())
		saveDialog.SetFileName(file.Name())
		saveDialog.SetLocation(ctx.Vm().GetLastSaveDir())
		saveDialog.Show()
	}
}
