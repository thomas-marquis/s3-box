package views

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	fyne_widget "fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
)

func GetNotificationView(appCtx appcontext.AppContext) (*fyne.Container, error) {
	notifications := appCtx.NotificationViewModel().Notifications()
	notificationList := fyne_widget.NewListWithData(
		notifications,
		func() fyne.CanvasObject {
			label := widget.NewOpenableLabel("", appCtx.Window())
			label.Label.Alignment = fyne.TextAlignLeading
			label.Label.Truncation = fyne.TextTruncateEllipsis
			label.Label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Label.Selectable = false
			return label
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			item, _ := di.(binding.Item[notification.Notification])
			notif, err := item.Get()
			if err != nil {
				panic(err)
			}
			formattedDt := notif.Time().Format("2006-01-02 15:04:05")
			var title, detail string
			switch notif.Type() {
			case notification.LevelError:
				title = fmt.Sprintf("%s: Error: %s", //nolint:errcheck
					formattedDt, notif.(notification.ErrorNotification).Error().Error())
			case notification.LevelInfo:
				title = fmt.Sprintf("%s: Info: %s", //nolint:errcheck
					formattedDt, notif.(notification.LogNotification).Title())
				detail = notif.(notification.LogNotification).Message()
			case notification.LevelDebug:
				title = fmt.Sprintf("%s: Debug: %s", //nolint:errcheck
					formattedDt, notif.(notification.LogNotification).Title())
				detail = notif.(notification.LogNotification).Message()
			}
			label := obj.(*widget.OpenableLabel)
			label.Label.Bind(binding.BindString(&title))
			label.Detail.Bind(binding.BindString(&detail))
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
