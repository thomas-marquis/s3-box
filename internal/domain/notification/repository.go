package notification

type Repository interface {
	Subscribe(channel chan Notification)
	Unsubscribe(channel chan Notification)
	Notify(notification Notification)
	NotifyError(err error)
	NotifyInfo(message string)
}
