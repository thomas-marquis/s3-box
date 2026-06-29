package notification

type Repository interface {
	Subscribe(channel chan Notification)
	Unsubscribe(channel chan Notification)
	Notify(notification Notification)
	NotifyError(err error)
	NotifyInfo(title, message string)
	NotifyDebug(title, message string)
}
