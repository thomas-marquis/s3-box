package connectionview

import (
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/connection"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

type connectionLine struct{}

func newConnectionLine() *connectionLine {
	return &connectionLine{}
}

func (*connectionLine) Raw() *fyne.Container {
	selected := widget.NewButtonWithIcon("", theme.RadioButtonIcon(), func() {})

	name := widget.NewLabel("")
	bucket := widget.NewLabel("")
	left := container.NewHBox(selected, name, widget.NewLabel("-"), bucket)

	edtiBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {})
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
	buttons := container.NewHBox(edtiBtn, deleteBtn)
	return container.NewBorder(
		nil, nil,
		left, buttons,
	)
}

func (*connectionLine) Update(ctx appcontext.AppContext, o fyne.CanvasObject, conn *connection.Connection) {
	c, _ := o.(*fyne.Container)

	leftGroup := c.Objects[0].(*fyne.Container)
	selected := leftGroup.Objects[0].(*widget.Button)
	if conn.IsSelected {
		selected.SetIcon(theme.RadioButtonCheckedIcon())
	} else {
		selected.SetIcon(theme.RadioButtonIcon())
	}
	selected.OnTapped = func() {
		if conn.IsSelected {
			return
		}
		dialog.ShowConfirm(
			"Select connection",
			fmt.Sprintf("Are you sure you want to select the connection '%s'?", conn.Name),
			func(confirmed bool) {
				if !confirmed {
					return
				}
				if err := ctx.ConnectionVM().SelectConnection(conn); err != nil {
					ctx.Log().Error("Failed to select connection", zap.Error(err))
				}
				if err := ctx.ConnectionVM().RefreshConnections(); err != nil {
					ctx.Log().Error("Failed to refresh connections", zap.Error(err))
				}
				if err := ctx.ExplorerVM().ResetTree(); err != nil {
					ctx.Log().Error("Failed to reset tree after connection changed", zap.Error(err))
				}
			},
			ctx.W(),
		)
	}

	name := leftGroup.Objects[1].(*widget.Label)
	name.SetText(conn.Name)

	bucket := leftGroup.Objects[3].(*widget.Label)
	bucket.SetText(fmt.Sprintf("%s/%s", conn.Server, conn.BucketName))

	btnGroup := c.Objects[1].(*fyne.Container)
	editBtn := btnGroup.Objects[0].(*widget.Button)
	editBtn.OnTapped = func() {
		newConnectionDialog(
			ctx, "Edit connection",
			conn.Name, conn.AccessKey, conn.SecretKey, conn.Server, conn.BucketName, conn.Region, conn.UseTls,
			true,
			func(name, accessKey, secretKey, server, bucket, region string, useTLS bool) error {
				conn.Name = name
				conn.AccessKey = accessKey
				conn.SecretKey = secretKey
				conn.Server = server
				conn.BucketName = bucket
				conn.Region = region
				conn.UseTls = useTLS
				return ctx.ConnectionVM().SaveConnection(conn)
			}).Show()
	}

	deleteBtn := btnGroup.Objects[1].(*widget.Button)
	deleteBtn.OnTapped = func() {
		dialog.ShowConfirm("Delete connection", fmt.Sprintf("Are you sure you want to delete the connection '%s'?", conn.Name), func(b bool) {
			if b {
				ctx.ConnectionVM().DeleteConenction(conn)
			}
		}, ctx.W())
	}
}
