package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
)

func GetNotificationView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	notificationList := widget.NewListWithData(
		appCtx.NotificationViewModel().Notifications(),
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			value, _ := di.(binding.String).Get()
			obj.(*widget.Label).SetText(value)
		},
	)

	return container.NewBorder(
		widget.NewLabel("Notifications"),
		nil, nil, nil,
		notificationList,
	), nil
}
