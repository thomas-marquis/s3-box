package viewmodel

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

type NotificationViewModel interface {
	Notifications() binding.List[notification.Notification]
}

type notificationViewModelImpl struct {
	notifications binding.List[notification.Notification]
	notifier      notification.Repository
}

func NewNotificationViewModel(notifier notification.Repository, terminated <-chan struct{}) NotificationViewModel {
	notifications := binding.NewList[notification.Notification](func(n notification.Notification, n2 notification.Notification) bool {
		return n.Id() == n2.Id()
	})
	notifStream := make(chan notification.Notification)

	notifier.Subscribe(notifStream)

	go func() {
		for {
			select {
			case <-terminated:
				return
			case notif := <-notifStream:
				notifications.Prepend(notif) //nolint:errcheck
			}
		}
	}()

	return &notificationViewModelImpl{
		notifications: notifications,
		notifier:      notifier,
	}
}

func (vm *notificationViewModelImpl) Notifications() binding.List[notification.Notification] {
	return vm.notifications
}
