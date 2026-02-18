package app

import (
	"fmt"
	"sync"

	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	// publicationWorkers defines the number of concurrent worker goroutines responsible for managing app events.
	publicationWorkers = 16

	// pubChanBufferSize defines the size of the channel used to publish events.
	// Increase this value to manage more subscribers without blocking event publishing.
	pubChanBufferSize = 100
)

type publishedLoad struct {
	evt              event.Event
	subscriberChanel chan event.Event
}

type eventBusImpl struct {
	sync.Mutex

	subscribers    map[chan event.Event]*event.Subscriber
	publishingChan chan publishedLoad
	done           <-chan struct{}
	notifier       notification.Repository
}

func newEventBusImpl(done <-chan struct{}, notifier notification.Repository) event.Bus {
	b := &eventBusImpl{
		subscribers:    make(map[chan event.Event]*event.Subscriber),
		publishingChan: make(chan publishedLoad, pubChanBufferSize),
		done:           done,
		notifier:       notifier,
	}

	for i := 0; i < publicationWorkers; i++ {
		go b.pubWorker()
	}

	go b.terminate()

	return b
}

func (b *eventBusImpl) pubWorker() {
	for {
		select {
		case <-b.done:
			return
		case i := <-b.publishingChan:
			select {
			case i.subscriberChanel <- i.evt:
			case <-b.done:
			}
		}
	}
}

func (b *eventBusImpl) terminate() {
	<-b.done
	for subChanel := range b.subscribers {
		close(subChanel)
	}
}

func (b *eventBusImpl) Subscribe() *event.Subscriber {
	b.Lock()
	defer b.Unlock()

	events := make(chan event.Event)
	subscriber := event.NewSubscriber(events)
	b.subscribers[events] = subscriber
	return subscriber
}

func (b *eventBusImpl) Publish(evt event.Event) {
	b.notifier.NotifyDebug(fmt.Sprintf("publishing event: %s", evt.Type()))
	b.Lock()
	defer b.Unlock()

	for channel, subscriber := range b.subscribers {
		if !subscriber.Accept(evt) {
			continue
		}
		select {
		case b.publishingChan <- publishedLoad{evt, channel}:
		case <-b.done:
		}
	}
	b.notifier.NotifyDebug(fmt.Sprintf("Published event: %s", evt.Type()))
}
