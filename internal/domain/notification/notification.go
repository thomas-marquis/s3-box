package notification

type Type string

const (
	Error Type = "notification.error"
	Info  Type = "notification.info"
)

type Notification interface {
	Type() Type
}

type ErrorNotification interface {
	Notification
	Error() error
}

type LogNotification interface {
	Notification
	Message() string
}

type errorNotificationImpl struct {
	err error
}

func NewError(err error) ErrorNotification {
	return errorNotificationImpl{err: err}
}

func (n errorNotificationImpl) Type() Type {
	return Error
}

func (n errorNotificationImpl) Error() error {
	return n.err
}

type infoNotificationImpl struct {
	message string
}

func NewInfo(message string) LogNotification {
	return infoNotificationImpl{message: message}
}

func (n infoNotificationImpl) Type() Type {
	return Info
}

func (n infoNotificationImpl) Message() string {
	return n.message
}
