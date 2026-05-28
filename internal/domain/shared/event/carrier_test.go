package event_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

type fakePayload string

func (p fakePayload) Type() event.Type {
	return event.Type("fake")
}

func TestCarriesAll_Dispatch(t *testing.T) {
	t.Run("should dispatch all carried events then the done event", func(t *testing.T) {
		// Given
		busDone := make(chan struct{})
		defer close(busDone)
		bus := event.NewInMemoryBus(busDone, &event.NopNotifier{})

		e1 := event.New(fakePayload("e1"))
		e1Res := event.NewFollowup(e1, fakePayload2{})

		e2 := event.New(fakePayload("e2"))
		e2Res := event.NewFollowup(e2, fakePayload("e2-success"))

		e3 := event.New(fakePayload("e3"))
		e3Res := event.NewFollowup(e3, fakePayload("e3-success"))

		ed := event.New(fakePayload("done"))

		c := event.NewCarriesAll(
			[]event.Event{e1, e2, e3},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
		)

		done := make(chan struct{})
		e1Done := make(chan struct{})
		e2Done := make(chan struct{})
		e3Done := make(chan struct{})

		testSub := bus.Subscribe()
		testSub.On(event.IsAny(), func(e event.Event) {
			switch e {
			case e1:
				bus.Publish(e1Res)
				close(e1Done)
			case e2:
				bus.Publish(e2Res)
				close(e2Done)
			case e3:
				bus.Publish(e3Res)
				close(e3Done)
			case ed:
				close(done)
			}
		}).ListenWithWorkers(1)
		defer testSub.Detach()

		// When
		bus.Publish(c)

		// Then
		testutil.AssertEventually(t, done)
		testutil.AssertEventually(t, e1Done)
		testutil.AssertEventually(t, e2Done)
		testutil.AssertEventually(t, e3Done)
	})

	t.Run("should dispatch all carried events then the done event with custom done condition", func(t *testing.T) {
		// Given
		busDone := make(chan struct{})
		defer close(busDone)
		bus := event.NewInMemoryBus(busDone, &event.NopNotifier{})

		in := event.New(fakePayload("e1"))
		eRes1 := event.NewFollowup(in, fakePayload("val1"))
		eRes2 := event.NewFollowup(in, fakePayload("val2"))

		ed := event.New(fakePayload("done"))

		c := event.NewCarriesAll(
			[]event.Event{in},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
			event.WithDoneCondition(func(sent, received event.Event) bool {
				return received.Payload.(fakePayload) == "val2"
			}),
		)

		done := make(chan struct{})
		eResDone := make(chan struct{})

		testSub := bus.Subscribe()
		testSub.On(event.IsAny(), func(e event.Event) {
			fmt.Println(e)
			switch e {
			case in:
				bus.Publish(eRes1)
				bus.Publish(eRes2)
				close(eResDone)
			case ed:
				close(done)
			}
		}).ListenWithWorkers(1)
		defer testSub.Detach()

		// When
		bus.Publish(c)

		// Then
		testutil.AssertEventually(t, done)
		testutil.AssertEventually(t, eResDone)
	})

	t.Run("should abort the dispatch and dispatch the timeout event process on timeout", func(t *testing.T) {
		// Given
		busDone := make(chan struct{})
		defer close(busDone)
		bus := event.NewInMemoryBus(busDone, &event.NopNotifier{})

		e1 := event.New(fakePayload("e1"))
		et := event.New(fakePayload("timeout"))

		c := event.NewCarriesAll(
			[]event.Event{e1},
			func([]event.Event) event.Event {
				return event.Event{}
			},
			et,
			event.WithTimeout(100*time.Millisecond),
		)

		timeoutReceived := make(chan struct{})

		testSub := bus.Subscribe()
		testSub.On(event.IsAny(), func(e event.Event) {
			if e == et {
				close(timeoutReceived)
			}
		}).ListenWithWorkers(1)
		defer testSub.Detach()

		// When
		bus.Publish(c)

		// Then
		testutil.AssertEventually(t, timeoutReceived)
	})
}

func TestCarriesSequence_Dispatch(t *testing.T) {
	t.Run("should dispatch all carried events sequentially then the done event", func(t *testing.T) {
		// Given
		busDone := make(chan struct{})
		defer close(busDone)
		bus := event.NewInMemoryBus(busDone, &event.NopNotifier{})

		e1 := event.New(fakePayload("e1"))
		e1Res := event.NewFollowup(e1, fakePayload2{})

		e2 := event.New(fakePayload("e2"))
		e2Res := event.NewFollowup(e2, fakePayload("e2-success"))

		e3 := event.New(fakePayload("e3"))
		e3Res := event.NewFollowup(e3, fakePayload("e3-success"))

		ed := event.New(fakePayload("done"))

		c := event.NewCarriesAll(
			[]event.Event{e1, e2, e3},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
		)

		done := make(chan struct{})
		e1Done := make(chan struct{})
		e2Done := make(chan struct{})
		e3Done := make(chan struct{})

		testSub := bus.Subscribe()
		testSub.On(event.IsAny(), func(e event.Event) {
			switch e {
			case e1:
				bus.Publish(e1Res)
				close(e1Done)
			case e2:
				bus.Publish(e2Res)
				select {
				case _, ok := <-e1Done:
					assert.False(t, ok)
				}
				close(e2Done)
			case e3:
				bus.Publish(e3Res)
				close(e3Done)
			case ed:
				close(done)
			}
		}).ListenWithWorkers(1)
		defer testSub.Detach()

		// When
		bus.Publish(c)

		// Then
		testutil.AssertEventually(t, done)
		testutil.AssertEventually(t, e1Done)
		testutil.AssertEventually(t, e2Done)
		testutil.AssertEventually(t, e3Done)
	})
}
