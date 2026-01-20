package app

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

const (
	publicationWorkers = 5
)

type publishedLoad struct {
	evt              event.Event
	subscriberChanel chan event.Event
}

type eventBusImpl struct {
	subscribers    map[chan event.Event]struct{}
	publishingChan chan publishedLoad
	done           <-chan struct{}
}

func NewEventBus(done <-chan struct{}) event.Bus {
	return newEventBusImpl(done)
}

func newEventBusImpl(done <-chan struct{}) event.Bus {
	b := &eventBusImpl{
		subscribers:    make(map[chan event.Event]struct{}),
		publishingChan: make(chan publishedLoad),
		done:           done,
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

func (b *eventBusImpl) Subscribe() <-chan event.Event {
	channel := make(chan event.Event)
	b.subscribers[channel] = struct{}{}
	return channel
}

func (b *eventBusImpl) Publish(evt event.Event) {
	for subscriber := range b.subscribers {
		select {
		case b.publishingChan <- publishedLoad{evt, subscriber}:
		case <-b.done:
		}
	}
}
