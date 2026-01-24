package notification

import "time"

type Level string

func (l Level) String() string {
	return string(l)
}

func (l Level) LowerOrEqual(target Level) bool {
	switch target {
	case LevelDebug:
		return true
	case LevelInfo:
		return l == LevelInfo || l == LevelError
	case LevelError:
		return l == LevelError
	}
	return false
}

const (
	LevelError Level = "notification.level.error"
	LevelInfo  Level = "notification.level.info"
	LevelDebug Level = "notification.level.debug"
)

type Notification interface {
	Type() Level
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

func (n errorNotificationImpl) Type() Level {
	return LevelError
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

func (n infoNotificationImpl) Type() Level {
	return LevelInfo
}

func (n infoNotificationImpl) Message() string {
	return n.message
}

type debugNotificationImpl struct {
	baseNotification
	message string
}

func NewDebug(message string) LogNotification {
	return debugNotificationImpl{baseNotification: newBaseNotification(), message: message}
}

func (n debugNotificationImpl) Type() Level {
	return LevelDebug
}

func (n debugNotificationImpl) Message() string {
	return n.message
}
