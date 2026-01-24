package infrastructure

import "github.com/thomas-marquis/s3-box/internal/domain/notification"

type notificationPublisher struct {
	subscribersSet map[chan notification.Notification]struct{}
	level          notification.Level
}

func NewNotificationPublisher(level notification.Level) notification.Repository {
	return &notificationPublisher{
		subscribersSet: make(map[chan notification.Notification]struct{}),
		level:          level,
	}
}

func (p *notificationPublisher) Subscribe(channel chan notification.Notification) {
	p.subscribersSet[channel] = struct{}{}
}

func (p *notificationPublisher) Unsubscribe(channel chan notification.Notification) {
	delete(p.subscribersSet, channel)
}

func (p *notificationPublisher) Notify(notification notification.Notification) {
	go func() {
		for channel := range p.subscribersSet {
			channel <- notification
		}
	}()
}

func (p *notificationPublisher) NotifyError(err error) {
	if notification.LevelError.LowerOrEqual(p.level) {
		p.Notify(notification.NewError(err))
	}
}

func (p *notificationPublisher) NotifyInfo(message string) {
	if notification.LevelInfo.LowerOrEqual(p.level) {
		p.Notify(notification.NewInfo(message))
	}
}

func (p *notificationPublisher) NotifyDebug(message string) {
	if notification.LevelDebug.LowerOrEqual(p.level) {
		p.Notify(notification.NewDebug(message))
	}
}
