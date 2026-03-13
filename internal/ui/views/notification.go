package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	fyne_widget "fyne.io/fyne/v2/widget"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

func GetNotificationView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	notifications := appCtx.NotificationViewModel().Notifications()
	notificationList := fyne_widget.NewListWithData(
		notifications,
		func() fyne.CanvasObject {
			label := widget.NewOpenableLabel("", appCtx.Window())
			label.Alignment = fyne.TextAlignLeading
			label.Truncation = fyne.TextTruncateEllipsis
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Selectable = false
			return label
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			value, _ := di.(binding.String)
			label := obj.(*widget.OpenableLabel)
			label.Bind(value)
		},
	)

	nothingToDisplay := container.NewCenter(
		fyne_widget.NewLabelWithStyle("Noting to display at the moment...",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true}))
	notificationList.Hide()

	notifications.AddListener(binding.NewDataListener(func() {
		if notifications.Length() == 0 {
			notificationList.Hide()
			nothingToDisplay.Show()
		} else {
			notificationList.Show()
			nothingToDisplay.Hide()
		}
	}))

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeading("Notifications"),
			fyne_widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(
			notificationList,
		),
		nothingToDisplay,
	), nil
}
