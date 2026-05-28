package event

import "time"

type TestHarness struct {
	bus     Bus
	timeout time.Duration
}

func NewTestHarness(bus Bus) *TestHarness {
	return &TestHarness{bus: bus, timeout: time.Second * 10}
}

func (h *TestHarness) Publish(evt Event) {

}

func (h *TestHarness) Expect(marble string, evtMap map[string]Payload) *TestHarness {
	return h
}

func (h *TestHarness) Given(marble string, evtMap map[string]Payload) *TestHarness {
	return h
}

func (h *TestHarness) PrayAndWait() {

}

type fakePayload string

func (f fakePayload) Type() Type { return "fakePayload" }

func Temp() {
	expected := "         'e1'____'e2'--____'e3'____d "
	completionEvents := " ____'r1'____--'r2'____'r3'- "
	evtMap := map[string]Payload{
		"e1": fakePayload("e1"),
		"e2": fakePayload("e2"),
		"e3": fakePayload("e3"),
		"d":  fakePayload("done"),
	}
	completionEventsMap := map[string]Payload{
		"r1": fakePayload("result from e1"),
		"r2": fakePayload("result from e2"),
		"r3": fakePayload("result from e3"),
	}

	done := make(chan struct{})
	bus := NewInMemoryBus(done, nil)
	th := NewTestHarness(bus).
		Given(completionEvents, completionEventsMap).
		Expect(expected, evtMap)

	inEvt := NewCarriesSequence(
		[]Event{
			New(fakePayload("e1")),
			New(fakePayload("e2")),
			New(fakePayload("e3")),
		},
		func(received []Event) Event {
			return New(fakePayload("done"))
		},
		New(fakePayload("timeout")),
	)
	th.Publish(inEvt)

	th.PrayAndWait()
}
