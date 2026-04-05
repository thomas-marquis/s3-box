package event

import "strings"

const (
	CarrierTypePrefix = "__carrier__"
)

func (t Type) IsCarrier() bool {
	return strings.HasPrefix(t.String(), CarrierTypePrefix)
}

type Carrier interface {
	Event

	// Dispatch dispatches all events in the carrier to the given channel.
	// This method is not supposed to be blocking.
	Dispatch(events chan Event)
}

type CarriesAll struct {
	Carried []Event
	Done    Event
}

func (c *CarriesAll) Dispatch(events chan Event) {
	panic("Not implemented")
	//go func() {
	//
	//}()
	//
	//go func() {
	//
	//}()
}
