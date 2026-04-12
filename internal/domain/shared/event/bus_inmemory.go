package event

import (
	"sync"
)

const (
	// publicationWorkers defines the number of concurrent worker goroutines responsible for managing app events.
	publicationWorkers = 16

	// pubChanBufferSize defines the size of the channel used to publish events.
	// Increase this value to manage more subscribers without blocking event publishing.
	pubChanBufferSize = 100
)

type publishedLoad struct {
	evt              Event
	subscriberChanel chan Event
}

type inMemoryBus struct {
	sync.Mutex

	subscribers    map[chan Event]*Subscriber
	publishingChan chan publishedLoad
	done           <-chan struct{}
	notifier       Notifier
}

func NewInMemoryBus(done <-chan struct{}, notifier Notifier) Bus {
	b := &inMemoryBus{
		subscribers:    make(map[chan Event]*Subscriber),
		publishingChan: make(chan publishedLoad, pubChanBufferSize),
		done:           done,
	}

	if notifier != nil {
		b.notifier = notifier
	} else {
		b.notifier = &NopNotifier{}
	}

	for i := 0; i < publicationWorkers; i++ {
		go b.pubWorker()
	}

	go b.terminate()

	return b
}

func (b *inMemoryBus) Subscribe() *Subscriber {
	b.Lock()
	defer b.Unlock()

	events := make(chan Event)
	subscriber := NewSubscriberWithBus(events, b)
	b.subscribers[events] = subscriber
	return subscriber
}

func (b *inMemoryBus) Publish(evt Event) {
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
	b.notifier.Notify(evt)
}

func (b *inMemoryBus) pubWorker() {
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

func (b *inMemoryBus) terminate() {
	<-b.done
	for subChanel := range b.subscribers {
		close(subChanel)
	}
}
