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
	notificationList := fyne_widget.NewListWithData(
		appCtx.NotificationViewModel().Notifications(),
		func() fyne.CanvasObject {
			return fyne_widget.NewTextGridFromString("")
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			value, _ := di.(binding.String).Get()
			obj.(*fyne_widget.TextGrid).SetText(value)
		},
	)

	return container.NewBorder(
		container.NewVBox(
			widget.NewHeading("Notifications"),
			fyne_widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(
			notificationList,
		),
	), nil
}
