package infrastructure

import "github.com/thomas-marquis/s3-box/internal/domain/notification"

type notificationPublisher struct {
	subscribersSet map[chan notification.Notification]struct{}
}

func NewNotificationPublisher() notification.Repository {
	return &notificationPublisher{
		subscribersSet: make(map[chan notification.Notification]struct{}),
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

func (p *notificationPublisher) NotifyError(err error) error {
	p.Notify(notification.NewError(err))
	return err
}

func (p *notificationPublisher) NotifyInfo(message string) {
	p.Notify(notification.NewInfo(message))
}
