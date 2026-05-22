package notification

import (
	"time"

	"github.com/google/uuid"
)

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
	Id() string
	Type() Level
	Time() time.Time
	Title() string
}

type baseNotification struct {
	id    string
	time  time.Time
	title string
}

func newBaseNotification(title string) baseNotification {
	return baseNotification{time: time.Now(), title: title, id: uuid.New().String()}
}

func (n baseNotification) Time() time.Time {
	return n.time
}

func (n baseNotification) Title() string {
	return n.title
}

func (n baseNotification) Id() string {
	return n.id
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
	return errorNotificationImpl{baseNotification: newBaseNotification("Error"), err: err}
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

func NewInfo(title, message string) LogNotification {
	return infoNotificationImpl{baseNotification: newBaseNotification(title), message: message}
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

func NewDebug(title, message string) LogNotification {
	return debugNotificationImpl{baseNotification: newBaseNotification(title), message: message}
}

func (n debugNotificationImpl) Type() Level {
	return LevelDebug
}

func (n debugNotificationImpl) Message() string {
	return n.message
}
