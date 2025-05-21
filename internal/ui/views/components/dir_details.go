package components

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/explorer"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type DirDetials struct {
	c *fyne.Container

	pathLabel      *widget.Label
	uploadBtn      *widget.Button
	newEmptyDirBtn *widget.Button
}

func NewDirDetails() *DirDetials {
	pathLabel := widget.NewLabel("")

	uploadBtn := widget.NewButton("Upload file", func() {})

	newEmptyDirBtn := widget.NewButtonWithIcon("New sub directory", theme.ContentAddIcon(), func() {})

	top := container.NewHBox(
		widget.NewIcon(theme.FolderIcon()),
		pathLabel,
	)
	c := container.NewBorder(
		top, container.NewVBox(uploadBtn, newEmptyDirBtn),
		nil, nil,
	)
	return &DirDetials{
		c:              c,
		pathLabel:      pathLabel,
		uploadBtn:      uploadBtn,
		newEmptyDirBtn: newEmptyDirBtn,
	}
}

func (d *DirDetials) Object() fyne.CanvasObject {
	return d.c
}

func (d *DirDetials) Update(ctx appcontext.AppContext, dir *explorer.S3Directory) {
	d.pathLabel.SetText(dir.ID.String())

	d.uploadBtn.OnTapped = func() {
		selectDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, ctx.Window()) // TODO better error handling
				return
			}

			if reader == nil {
				return
			}

			localDestFilePath := reader.URI().Path()
			if err := ctx.ExplorerViewModel().UploadFile(localDestFilePath, dir); err != nil {
				dialog.ShowError(err, ctx.Window()) // TODO better error handling
				return
			}
			if err := ctx.ExplorerViewModel().SetLastUploadDir(localDestFilePath); err != nil {
				dialog.ShowError(err, ctx.Window()) // TODO better error handling
				return
			}

			// if err := ctx.Vm().RefreshDir(dir); err != nil {
			// 	dialog.ShowError(err, ctx.W()) // TODO better error handling
			// 	return
			// }
			dialog.ShowInformation("Upload", "File uploaded", ctx.Window())
		}, ctx.Window())

		selectDialog.SetLocation(ctx.ExplorerViewModel().GetLastUploadDir())
		selectDialog.Show()
	}

	d.newEmptyDirBtn.OnTapped = func() {
		nameEntry := widget.NewEntry()
		dialog.ShowForm(
			fmt.Sprintf("New directory under %s", dir.Name),
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
				_, err := ctx.ExplorerViewModel().CreateEmptyDirectory(dir, name)
				if err != nil {
					dialog.ShowError(err, ctx.Window())
					return
				}
				if err := ctx.ExplorerViewModel().RefreshDir(dir.ID); err != nil {
					dialog.ShowError(err, ctx.Window())
					return
				}
			},
			ctx.Window(),
		)
	}
}
