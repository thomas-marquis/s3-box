package utils

type Publisher[T any] struct {
	subscribers []chan T
	done        <-chan struct{}
}

func NewPublisher[T any](done <-chan struct{}) *Publisher[T] {
	p := &Publisher[T]{
		subscribers: make([]chan T, 0),
		done:        done,
	}

	go func() {
		for {
			select {
			case _, ok := <-done:
				if !ok {
					for _, channel := range p.subscribers {
						close(channel)
					}
				}
				return
			}
		}
	}()

	return p
}

func (e *Publisher[T]) Subscribe() chan T {
	subscriber := make(chan T)
	e.subscribers = append(e.subscribers, subscriber)
	return subscriber
}

func (e *Publisher[T]) Publish(event T) {
	for _, subscriber := range e.subscribers {
		select {
		case subscriber <- event:
		case <-e.done:
		}
	}
}
