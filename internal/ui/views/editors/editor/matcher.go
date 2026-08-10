package editor

import "github.com/thomas-marquis/it-happened/event"

type forCurrentEditor struct {
	Editor Editor
}

func (m forCurrentEditor) Match(evt event.Event) bool {
	if pl, ok := evt.Payload().(Payload); ok {
		return pl.This() == m.Editor
	}
	return true // Let the editor subscribe to all application's other events
}
