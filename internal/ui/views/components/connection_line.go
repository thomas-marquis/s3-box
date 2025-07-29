package components

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

type ConnectionLine struct {
	deck *connection_deck.Deck
}

func NewConnectionLine(deck *connection_deck.Deck) *ConnectionLine {
	return &ConnectionLine{deck}
}

func (*ConnectionLine) Raw() *fyne.Container {
	selected := widget.NewButtonWithIcon("", theme.RadioButtonIcon(), func() {})

	name := widget.NewLabel("")
	bucket := widget.NewLabel("")
	left := container.NewHBox(selected, name, widget.NewLabel("-"), bucket)

	editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {})
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
	buttons := container.NewHBox(editBtn, deleteBtn)
	return container.NewBorder(
		nil, nil,
		left, buttons,
	)
}

func (l *ConnectionLine) Update(ctx appcontext.AppContext, o fyne.CanvasObject, conn *connection_deck.Connection) {
	c, _ := o.(*fyne.Container)

	leftGroup := c.Objects[0].(*fyne.Container)
	selected := leftGroup.Objects[0].(*widget.Button)
	if conn.Is(l.deck.SelectedConnection()) {
		selected.SetIcon(theme.RadioButtonCheckedIcon())
	} else {
		selected.SetIcon(theme.RadioButtonIcon())
	}
	selected.OnTapped = func() {
		if conn.Is(l.deck.SelectedConnection()) {
			return
		}
		dialog.ShowConfirm(
			"Select connection",
			fmt.Sprintf("Are you sure you want to select the connection '%s'?", conn.Name()),
			func(confirmed bool) {
				if !confirmed {
					return
				}

				if _, err := ctx.ConnectionViewModel().Select(conn); err != nil {
					ctx.L().Error("Failed to select connection", zap.Error(err))
					dialog.ShowError(err, ctx.Window())
				}
			},
			ctx.Window(),
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

	btnGroup := c.Objects[1].(*fyne.Container)
	editBtn := btnGroup.Objects[0].(*widget.Button)
	editBtn.OnTapped = func() {
		NewConnectionDialog(
			ctx,
			"Edit connection",
			*conn,
			true,
			func(name, accessKey, secretKey, bucket string, options ...connection_deck.ConnectionOption) error {
				opts := make([]connection_deck.ConnectionOption, 0, len(options)+4)
				opts = append(opts,
					connection_deck.WithName(name),
					connection_deck.WithCredentials(accessKey, secretKey),
					connection_deck.WithBucket(bucket),
				)
				opts = append(opts, options...)
				return ctx.ConnectionViewModel().Update(conn.ID(), opts...)
			}).Show()
	}

	deleteBtn := btnGroup.Objects[1].(*widget.Button)
	deleteBtn.OnTapped = func() {
		dialog.ShowConfirm("Delete connection", fmt.Sprintf("Are you sure you want to delete the connection '%s'?", conn.Name), func(b bool) {
			if b {
				if err := ctx.ConnectionViewModel().Delete(conn.ID()); err != nil {
					ctx.L().Error("Failed to delete connection", zap.Error(err))
				}
			}
		}, ctx.Window())
	}
}
