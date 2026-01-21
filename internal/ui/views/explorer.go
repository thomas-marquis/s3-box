package views

import (
	"errors"

	"fyne.io/fyne/v2/dialog"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"

	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"

	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	fyne_widget "fyne.io/fyne/v2/widget"
)

func makeNoConnectionTopBanner(ctx appcontext.AppContext) *fyne.Container {
	return container.NewVBox(
		container.NewCenter(fyne_widget.NewLabel("No connection selected, please select a connection in the settings menu")),
		container.NewCenter(fyne_widget.NewButton("Manage connections", func() {
			if _, err := ctx.Navigate(navigation.ConnectionRoute); err != nil { //nolint:staticcheck
				// TODO: handle error
			}
		})),
	)
}

// GetFileExplorerView initializes and returns the file explorer UI layout with functionality for file and directory navigation.
// It implements the navigation.View type interface.
// Returns filled the *fyne.Container and an error.
func GetFileExplorerView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	noConn := makeNoConnectionTopBanner(appCtx)
	noConn.Hide()
	vm := appCtx.ExplorerViewModel()

	headingData := binding.NewString()
	headingData.Set("File explorer") //nolint:errcheck

	content := container.NewHSplit(fyne_widget.NewLabel(""), fyne_widget.NewLabel(""))

	vm.SelectedConnection().AddListener(binding.NewDataListener(func() {
		conn := vm.CurrentSelectedConnection()
		if conn == nil {
			noConn.Show()
			content.Hide()
		} else {
			headingData.Set("File explorer: " + conn.Name()) //nolint:errcheck
			noConn.Hide()
			content.Show()
		}
	}))

	vm.ErrorMessage().AddListener(binding.NewDataListener(func() {
		msg, _ := vm.ErrorMessage().Get()
		if msg == "" {
			return
		}
		dialog.ShowError(errors.New(msg), appCtx.Window())
		vm.ErrorMessage().Set("") //nolint:errcheck
	}))

	vm.InfoMessage().AddListener(binding.NewDataListener(func() {
		msg, _ := vm.InfoMessage().Get()
		if msg == "" {
			return
		}
		dialog.ShowInformation("Info", msg, appCtx.Window())
		vm.InfoMessage().Set("") //nolint:errcheck
	}))

	detailsContainer := container.NewVBox()
	fileDetails := widget.NewFileDetails(appCtx)
	dirDetails := widget.NewDirectoryDetails(appCtx)

	tree := widget.NewExplorerTree(appCtx,
		func(dir *directory.Directory) {
			dirDetails.Render(dir)
			detailsContainer.Objects = []fyne.CanvasObject{dirDetails}
		},
		func(file *directory.File) {
			fileDetails.Render(file)
			detailsContainer.Objects = []fyne.CanvasObject{fileDetails}
		},
	)

	content.Leading = container.NewScroll(tree)
	content.Trailing = detailsContainer

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeadingWithData(headingData),
			fyne_widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewBorder(
			noConn,
			nil,
			nil,
			nil,
			content,
		),
	), nil
}
