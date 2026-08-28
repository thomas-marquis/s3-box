package views

import (
	"errors"

	"fyne.io/fyne/v2/dialog"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/u"

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
	st := appCtx.State()

	headingData := binding.NewString()
	u.Skip(headingData.Set("File explorer"))

	content := container.NewHSplit(fyne_widget.NewLabel(""), fyne_widget.NewLabel(""))

	st.Connection().Selected().AddListener(binding.NewDataListener(func() {
		conn := u.SkipV(st.Connection().Selected().Get())
		if conn == nil {
			noConn.Show()
			content.Hide()
		} else {
			val := "File explorer: " + conn.Name()
			if conn.ReadOnly() {
				val += " (read-only)"
			}
			u.Skip(headingData.Set(val))
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
		u.Skip(vm.ErrorMessage().Set(""))
	}))

	vm.InfoMessage().AddListener(binding.NewDataListener(func() {
		msg, _ := vm.InfoMessage().Get()
		if msg == "" {
			return
		}
		dialog.ShowInformation("Info", msg, appCtx.Window())
		u.Skip(vm.InfoMessage().Set(""))
	}))

	go func() {
		for evt := range vm.PendingUserValidations() {
			dialog.ShowConfirm("It's up to you!", evt.Message, func(validated bool) {
				vm.Validate(evt, validated)
			}, appCtx.Window())
		}
	}()

	fileDetails := widget.NewFileDetails(appCtx)
	fileDetails.Hide()

	dirDetails := widget.NewDirectoryDetails(appCtx)
	dirDetails.Hide()

	detailsContainer := container.NewStack(fileDetails, dirDetails)

	st.Explorer().SelectedDir().AddListener(binding.NewDataListener(func() {
		dir := u.SkipV(st.Explorer().SelectedDir().Get())
		if dir == nil {
			dirDetails.Hide()
			return
		}
		dirDetails.Select(dir)
		dirDetails.Show()
		fileDetails.Hide()
	}))

	st.Explorer().SelectedFile().AddListener(binding.NewDataListener(func() {
		file := u.SkipV(st.Explorer().SelectedFile().Get())
		if file == nil {
			fileDetails.Hide()
			return
		}
		fileDetails.Select(file)
		fileDetails.Show()
		dirDetails.Hide()
	}))

	tree := widget.NewExplorerTree(appCtx,
		func(dir *directory.Directory) {
			st.Explorer().SelectDir(dir)
		},
		func(file *directory.File) {
			st.Explorer().SelectFile(file)
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
