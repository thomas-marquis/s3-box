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
	wg             sync.WaitGroup
}

// NewInMemoryBus creates a new in-memory event bus.
// This implementation allows blocking carrier Dispatch method.
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
		b.wg.Add(1)
		go b.pubWorker()
	}

	go b.terminate()

	return b
}

func (b *inMemoryBus) Subscribe() *Subscriber {
	b.Lock()
	defer b.Unlock()

	events := make(chan Event)
	subscriber := NewSubscriber(events)
	b.subscribers[events] = subscriber
	return subscriber
}

func (b *inMemoryBus) Publish(evt Event) {
	b.notifier.Notify(evt)
	if c, ok := evt.Payload.(Carrier); ok {
		go c.Dispatch(b)
		return
	}

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
}

func (b *inMemoryBus) pubWorker() {
	defer b.wg.Done()
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
	b.wg.Wait()
	b.Lock()
	defer b.Unlock()
	for subChanel := range b.subscribers {
		close(subChanel)
	}
	clear(b.subscribers)
}
