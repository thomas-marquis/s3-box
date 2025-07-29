package notification

import "time"

type Type string

const (
	Error Type = "notification.error"
	Info  Type = "notification.info"
)

type Notification interface {
	Type() Type
	Time() time.Time
}

type baseNotification struct {
	time time.Time
}

func newBaseNotification() baseNotification {
	return baseNotification{time: time.Now()}
}

func (n baseNotification) Time() time.Time {
	return n.time
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
	baseNotification
	err error
}

func NewError(err error) ErrorNotification {
	return errorNotificationImpl{baseNotification: newBaseNotification(), err: err}
}

func (n errorNotificationImpl) Type() Type {
	return Error
}

func (n errorNotificationImpl) Error() error {
	return n.err
}

type infoNotificationImpl struct {
	baseNotification
	message string
}

func NewInfo(message string) LogNotification {
	return infoNotificationImpl{baseNotification: newBaseNotification(), message: message}
}

func (n infoNotificationImpl) Type() Type {
	return Info
}

func (n infoNotificationImpl) Message() string {
	return n.message
}
