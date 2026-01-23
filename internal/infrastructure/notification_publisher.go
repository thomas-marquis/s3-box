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
	if p.level.IsAtLeast(notification.LevelError) {
		p.Notify(notification.NewError(err))
	}
}

func (p *notificationPublisher) NotifyInfo(message string) {
	if p.level.IsAtLeast(notification.LevelInfo) {
		p.Notify(notification.NewInfo(message))
	}
}

func (p *notificationPublisher) NotifyDebug(message string) {
	if p.level.IsAtLeast(notification.LevelDebug) {
		p.Notify(notification.NewDebug(message))
	}
}
