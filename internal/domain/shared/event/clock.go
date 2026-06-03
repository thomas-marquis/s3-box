package event

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	AfterFunc(d time.Duration, f func()) Timer
}

type Timer interface {
	Stop() bool
}

type RealClock struct{}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

func (c *RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *RealClock) AfterFunc(d time.Duration, f func()) Timer {
	return &realTimer{time.AfterFunc(d, f)}
}

type realTimer struct {
	*time.Timer
}

func (t *realTimer) Stop() bool {
	return t.Timer.Stop()
}

type FakeClock struct {
	now    time.Time
	timers []*fakeTimer
	mu     sync.Mutex
}

type fakeTimer struct {
	ch       chan time.Time
	duration time.Duration
	callback func()
	stopped  bool
}

func NewFakeClock() *FakeClock {
	return &FakeClock{now: time.Now()}
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.timers = append(c.timers, &fakeTimer{ch: ch, duration: d})
	return ch
}

func (c *FakeClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	timer := &fakeTimer{duration: d, callback: f, ch: make(chan time.Time, 1)}
	c.timers = append(c.timers, timer)
	return timer
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	for _, t := range c.timers {
		if !t.stopped {
			t.duration -= d
			if t.duration <= 0 {
				if t.callback != nil {
					t.callback()
				}
				if t.ch != nil {
					t.ch <- c.now
				}
				t.stopped = true
			}
		}
	}
}

func (t *fakeTimer) Stop() bool {
	t.stopped = true
	return true
}
