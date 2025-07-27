package viewmodel

import "fyne.io/fyne/v2/data/binding"

type NotificationViewModel interface {
	SendError(error)
	Notifications() binding.StringList
}

type notificationViewModelImpl struct {
	notifications binding.StringList
	errorStream   chan error
}

func NewNotificationViewModel(errorStream chan error, terminate <-chan struct{}) NotificationViewModel {
	notifications := binding.NewStringList()

	go func() {
		for {
			select {
			case <-terminate:
				return
			case err := <-errorStream:
				notifications.Prepend(err.Error())
			}
		}
	}()

	return &notificationViewModelImpl{
		notifications: notifications,
		errorStream:   errorStream,
	}
}

func (vm *notificationViewModelImpl) SendError(err error) {
	go func() {
		vm.errorStream <- err
	}()
}

func (vm *notificationViewModelImpl) Notifications() binding.StringList {
	return vm.notifications
}
