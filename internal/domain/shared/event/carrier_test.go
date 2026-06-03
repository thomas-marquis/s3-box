package event_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/carrier"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
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
		bus := inmemory.NewBus(busDone, &event.NopNotifier{})

		e1 := event.New(fakePayload("e1"))
		e1Res := event.NewFollowup(e1, fakePayload2{})

		e2 := event.New(fakePayload("e2"))
		e2Res := event.NewFollowup(e2, fakePayload("e2-success"))

		e3 := event.New(fakePayload("e3"))
		e3Res := event.NewFollowup(e3, fakePayload("e3-success"))

		ed := event.New(fakePayload("done"))

		c := carrier.NewAll(
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

// =============================================================================
// TestHarness-based Tests for Carriers
// =============================================================================

func setupHarness(opts ...event.TestHarnessOption) (*event.TestHarness, chan struct{}) {
	busDone := make(chan struct{})
	bus := event.NewInMemoryBus(busDone, &event.NopNotifier{})
	return event.NewTestHarness(bus, opts...), busDone
}

func TestCarriesAll_WithHarness(t *testing.T) {
	t.Run("should dispatch all carried events then the done event", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		e2 := event.New(fakePayload("e2"))
		e3 := event.New(fakePayload("e3"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"e2": fakePayload("e2"),
			"e3": fakePayload("e3"),
			"r1": fakePayload("r1"),
			"r2": fakePayload("r2"),
			"r3": fakePayload("r3"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesAll(
			[]event.Event{e1, e2, e3},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
		)

		in := "   ______________('r1<-e1''r2<-e2')--'r3<-e3'    "
		exp := "  ('e1''e2''e3')('r1''r2')_________-'r3'    'ed'"

		// When
		h.Given(in, payloads)
		h.Expect(exp, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should dispatch all carried events then the done event with custom done condition", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		in := event.New(fakePayload("e1"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"in": fakePayload("e1"),
			"r1": fakePayload("val1"),
			"r2": fakePayload("val2"),
			"ed": fakePayload("done"),
		}

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

		given := " ('r1<-in''r2<-in')"
		expect := "('in''r1''r2')    'ed'"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should abort dispatch and publish timeout event on timeout", func(t *testing.T) {
		// Given
		h, busDone := setupHarness(event.WithTickDuration(20 * time.Millisecond))
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		et := event.New(fakePayload("timeout"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"et": fakePayload("timeout"),
		}

		c := event.NewCarriesAll(
			[]event.Event{e1},
			func([]event.Event) event.Event {
				return event.Event{}
			},
			et,
			event.WithTimeout(100*time.Millisecond),
		)

		expect := "'e1'----'et'"

		// When
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should handle single event carrier", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("single"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("single"),
			"r1": fakePayload("response"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesAll(
			[]event.Event{e1},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
		)

		given := " 'r1<-e1'      "
		expect := "('e1''r1')'ed'"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should handle concurrent events with max concurrency", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		e2 := event.New(fakePayload("e2"))
		e3 := event.New(fakePayload("e3"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"e2": fakePayload("e2"),
			"e3": fakePayload("e3"),
			"r1": fakePayload("r1"),
			"r2": fakePayload("r2"),
			"r3": fakePayload("r3"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesAll(
			[]event.Event{e1, e2, e3},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
			event.WithMaxConcurrency(3),
		)

		given := " ('r1<-e1''r2<-e2''r3<-e3')"
		expect := "('e1''e2''e3''r1''r2''r3')'ed'"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should handle multiple followups for same event", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"r1": fakePayload("response1"),
			"r2": fakePayload("response2"),
			"r3": fakePayload("response3"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesAll(
			[]event.Event{e1},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
			event.WithDoneCondition(func(sent, received event.Event) bool {
				return true
			}),
		)

		given := " ('r1<-e1''r2<-e1''r3<-e1')"
		expect := "('e1''r1''r2''r3')        'ed'"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should handle custom max concurrency", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		e2 := event.New(fakePayload("e2"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"e2": fakePayload("e2"),
			"r1": fakePayload("r1"),
			"r2": fakePayload("r2"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesAll(
			[]event.Event{e1, e2},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
			event.WithMaxConcurrency(1),
			event.WithTimeout(500*time.Millisecond),
		)

		given := " ('r1<-e1''r2<-e2')"
		expect := "('e1''e2''r1''r2')'ed'"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})
}

func TestCarriesSequence_WithHarness(t *testing.T) {
	t.Run("should dispatch all carried events sequentially then the done event", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		e2 := event.New(fakePayload("e2"))
		e3 := event.New(fakePayload("e3"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"e2": fakePayload("e2"),
			"e3": fakePayload("e3"),
			"r1": fakePayload("r1"),
			"r2": fakePayload("r2"),
			"r3": fakePayload("r3"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesSequence(
			[]event.Event{e1, e2, e3},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
		)

		given := " ('r1<-e1''r2<-e2''r3<-e3')"
		expect := "('e1''e2''e3''r1''r2''r3''ed')"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should handle single event in sequence", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("single"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("single"),
			"r1": fakePayload("response"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesSequence(
			[]event.Event{e1},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
		)

		given := " 'r1<-e1'"
		expect := "('e1''r1''ed')"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should handle empty sequence", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		c := event.NewCarriesSequence(
			[]event.Event{},
			func([]event.Event) event.Event {
				return event.Event{}
			},
			event.Event{},
		)

		// When
		h.Expect("", map[string]event.Payload{})
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})

	t.Run("should use custom done condition", func(t *testing.T) {
		// Given
		h, busDone := setupHarness()
		defer close(busDone)

		e1 := event.New(fakePayload("e1"))
		ed := event.New(fakePayload("done"))

		payloads := map[string]event.Payload{
			"e1": fakePayload("e1"),
			"r1": fakePayload("val1"),
			"r2": fakePayload("val2"),
			"ed": fakePayload("done"),
		}

		c := event.NewCarriesSequence(
			[]event.Event{e1},
			func([]event.Event) event.Event {
				return ed
			},
			event.Event{},
			event.WithDoneCondition(func(sent, received event.Event) bool {
				return received.Payload.(fakePayload) == "val2"
			}),
		)

		given := " ('r1<-e1''r2<-e1')"
		expect := "('e1''r1''r2''ed')"

		// When
		h.Given(given, payloads)
		h.Expect(expect, payloads)
		h.Publish(c)

		// Then
		if !h.PrayAndWait() {
			t.Errorf("TestHarness failed: %s", h.GetFailureReason())
		}
	})
}
