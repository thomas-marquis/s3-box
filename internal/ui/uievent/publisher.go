package uievent

import "fmt"

type Publisher interface {
	Publish(event UiEvent)
	Subscribe() <-chan UiEvent
}

type publisher struct {
	subscribersSet map[chan UiEvent]struct{}
}

func NewPublisher(done <-chan struct{}) Publisher {
	p := &publisher{
		subscribersSet: make(map[chan UiEvent]struct{}),
	}
	go func() {
		for {
			select {
			case _, ok := <-done:
				if !ok {
					for channel := range p.subscribersSet {
						close(channel)
					}
				}
				return
			}
		}
	}()
	return p
}

func (p *publisher) Publish(event UiEvent) {
	go func() {
		for channel := range p.subscribersSet {
			select {
			case channel <- event:
			default:
				fmt.Println("failed to broadcast event: ", event)
			}
		}
	}()
}

func (p *publisher) Subscribe() <-chan UiEvent {
	channel := make(chan UiEvent)
	p.subscribersSet[channel] = struct{}{}
	return channel
}
