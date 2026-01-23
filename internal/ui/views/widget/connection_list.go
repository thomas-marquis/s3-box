package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
)

type ConnectionList struct {
	widget.BaseWidget
	connections binding.UntypedList
	appCtx      appcontext.AppContext
}

func NewConnectionList(appCtx appcontext.AppContext) *ConnectionList {
	vm := appCtx.ConnectionViewModel()

	w := &ConnectionList{
		connections: vm.Connections(),
		appCtx:      appCtx,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ConnectionList) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)
	return widget.NewSimpleRenderer(widget.NewListWithData(
		w.connections,
		w.makeRowListItem,
		w.updateListItem,
	))
}

func (w *ConnectionList) makeRowListItem() fyne.CanvasObject {
	selected := widget.NewButtonWithIcon("", theme.RadioButtonIcon(), func() {})

	name := widget.NewLabel("")
	bucket := widget.NewLabel("")

	viewFilesBtn := widget.NewButtonWithIcon(
		"View files",
		theme.NavigateNextIcon(),
		func() {
			if _, err := w.appCtx.Navigate(navigation.ExplorerRoute); err != nil { //nolint:staticcheck
				dialog.ShowError(err, w.appCtx.Window())
			}
		},
	)
	viewFilesBtn.Hide()

	left := container.NewHBox(selected, name, widget.NewLabel("-"), bucket)

	editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {})
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
	buttons := container.NewHBox(viewFilesBtn, editBtn, deleteBtn)
	return container.NewBorder(
		nil, nil,
		left, buttons,
	)
}

func (w *ConnectionList) updateListItem(di binding.DataItem, o fyne.CanvasObject) {
	i, _ := di.(binding.Untyped).Get()
	conn, _ := i.(*connection_deck.Connection)
	vm := w.appCtx.ConnectionViewModel()

	c, _ := o.(*fyne.Container)

	leftGroup := c.Objects[0].(*fyne.Container)
	btnGroup := c.Objects[1].(*fyne.Container)

	selected := leftGroup.Objects[0].(*widget.Button)
	viewFilesBtn := btnGroup.Objects[0].(*widget.Button)

	if conn.Is(vm.Deck().SelectedConnection()) {
		selected.SetIcon(theme.RadioButtonCheckedIcon())
		viewFilesBtn.Show()
	} else {
		selected.SetIcon(theme.RadioButtonIcon())
		viewFilesBtn.Hide()
	}
	selected.OnTapped = func() {
		if conn.Is(vm.Deck().SelectedConnection()) {
			return
		}
		dialog.ShowConfirm(
			"Select connection",
			fmt.Sprintf("Are you sure you want to select the connection '%s'?", conn.Name()),
			func(confirmed bool) {
				if !confirmed {
					return
				}
				vm.Select(conn)
			},
			w.appCtx.Window(),
		)
	}

	name := leftGroup.Objects[1].(*widget.Label)
	if conn.ReadOnly() {
		name.SetText(fmt.Sprintf("%s (read-only)", conn.Name()))
	} else {
		name.SetText(conn.Name())
	}

	bucket := leftGroup.Objects[3].(*widget.Label)
	bucket.SetText(fmt.Sprintf("%s/%s", conn.Server(), conn.Bucket()))

	editBtn := btnGroup.Objects[1].(*widget.Button)
	editBtn.OnTapped = NewConnectionForm(w.appCtx, conn, true,
		func(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) {
			opts := make([]connection_deck.ConnectionOption, 0, len(options)+3)
			opts = append(opts,
				connection_deck.WithName(name),
				connection_deck.WithCredentials(accessKey, secretKey),
				connection_deck.WithBucket(bucket),
			)
			opts = append(opts, options...)
			vm.Update(conn.ID(), opts...)
		},
	).AsDialog("Edit connection").Show

	deleteBtn := btnGroup.Objects[2].(*widget.Button)
	deleteBtn.OnTapped = func() {
		dialog.ShowConfirm("Delete connection",
			fmt.Sprintf("Are you sure you want to delete the connection '%s'?", conn.Name()),
			func(confirmed bool) {
				if confirmed {
					vm.Delete(conn)
				}
			}, w.appCtx.Window())
	}
}
