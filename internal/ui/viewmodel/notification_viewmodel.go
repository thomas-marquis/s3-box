package viewmodel

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

type NotificationViewModel interface {
	Notifications() binding.StringList
	SendError(error)
	SendInfo(string)
}

type notificationViewModelImpl struct {
	notifications binding.StringList
	notifier      notification.Repository
}

func NewNotificationViewModel(notifier notification.Repository, terminated <-chan struct{}) NotificationViewModel {
	notifications := binding.NewStringList()
	notifStream := make(chan notification.Notification)

	notifier.Subscribe(notifStream)

	go func() {
		for {
			select {
			case <-terminated:
				return
			case notif := <-notifStream:
				formattedDt := notif.Time().Format("2006-01-02 15:04:05")
				switch notif.Type() {
				case notification.Error:
					notifications.Prepend(fmt.Sprintf("%s: Error: %s",
						formattedDt, notif.(notification.ErrorNotification).Error().Error()))
				case notification.Info:
					notifications.Prepend(fmt.Sprintf("%s: Info: %s",
						formattedDt, notif.(notification.LogNotification).Message()))
				}
			}
		}
	}()

	return &notificationViewModelImpl{
		notifications: notifications,
		notifier:      notifier,
	}
}

func (vm *notificationViewModelImpl) Notifications() binding.StringList {
	return vm.notifications
}

func (vm *notificationViewModelImpl) SendError(err error) {
	vm.notifier.NotifyError(err)
}

func (vm *notificationViewModelImpl) SendInfo(msg string) {
	vm.notifier.NotifyInfo(msg)
}
