package explorerview

import (
	"github.com/thomas-marquis/s3-box/internal/explorer"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type dirDetials struct {
	c *fyne.Container

	pathLabel *widget.Label
	uploadBtn *widget.Button
}

func newDirDetails() *dirDetials {
	pathLabel := widget.NewLabel("")

	uploadBtn := widget.NewButton("Upload file", func() {})

	top := container.NewHBox(
		widget.NewIcon(theme.FolderIcon()),
		pathLabel,
	)
	c := container.NewBorder(
		top, uploadBtn,
		nil, nil,
	)
	return &dirDetials{
		c:         c,
		pathLabel: pathLabel,
		uploadBtn: uploadBtn,
	}
}

func (d *dirDetials) Object() fyne.CanvasObject {
	return d.c
}

func (d *dirDetials) Update(ctx appcontext.AppContext, dir *explorer.Directory) {
	d.pathLabel.SetText(dir.Path())

	d.uploadBtn.OnTapped = func() {
		selectDialog := dialog.NewFileOpen(makeHandleOnUploadTapped(ctx, dir), ctx.W())
		selectDialog.SetLocation(ctx.ExplorerVM().GetLastUploadDir())
		selectDialog.Show()
	}
}

func makeHandleOnUploadTapped(ctx appcontext.AppContext, dir *explorer.Directory) func(reader fyne.URIReadCloser, err error) {
	return func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, ctx.W()) // TODO better error handling
			return
		}

		if reader == nil {
			return
		}

		localDestFilePath := reader.URI().Path()
		if err := ctx.ExplorerVM().UploadFile(localDestFilePath, dir); err != nil {
			dialog.ShowError(err, ctx.W()) // TODO better error handling
			return
		}
		if err := ctx.ExplorerVM().SetLastUploadDir(localDestFilePath); err != nil {
			dialog.ShowError(err, ctx.W()) // TODO better error handling
			return
		}

		if err := ctx.ExplorerVM().RefreshDir(dir); err != nil {
			dialog.ShowError(err, ctx.W()) // TODO better error handling
			return
		}
		dialog.ShowInformation("Upload", "File uploaded", ctx.W())
	}

}
