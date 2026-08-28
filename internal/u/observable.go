package u

import (
	"sync"
)

type Observable[T any] struct {
	observers  map[string]map[int]func(value T)
	mu         sync.RWMutex
	currObsIdx int
}

func NewObservable[T any]() *Observable[T] {
	return &Observable[T]{
		observers: make(map[string]map[int]func(value T)),
	}
}

func (o *Observable[T]) ObserveWithName(name string, f func(value T)) func() {
	o.mu.Lock()
	defer o.mu.Unlock()
	defer func() { o.currObsIdx++ }()

	if _, ok := o.observers[name]; !ok {
		o.observers[name] = make(map[int]func(value T))
	}
	o.observers[name][o.currObsIdx] = f
	currIdx := o.currObsIdx

	return func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		if funcs, found := o.observers[name]; found {
			if _, stillHere := funcs[currIdx]; stillHere {
				delete(o.observers[name], currIdx)
			}
		}
	}
}

func (o *Observable[T]) Observe(f func(value T)) func() {
	return o.ObserveWithName("", f)
}

func (o *Observable[T]) TriggerAll(value T) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, observers := range o.observers {
		for _, observer := range observers {
			observer(value)
		}
	}
}

func (o *Observable[T]) TriggerWithName(name string, value T) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if observers, ok := o.observers[name]; ok {
		for _, observer := range observers {
			observer(value)
		}
	}
}

func (o *Observable[T]) RemoveObserversWithName(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.observers, name)
}

func (o *Observable[T]) RemoveAllObservers() {
	o.mu.Lock()
	defer o.mu.Unlock()
	clear(o.observers)
}

type ObservableValue[T any] struct {
	*Observable[T]
	value T
}

func NewObservableValue[T any](initialValue T) *ObservableValue[T] {
	return &ObservableValue[T]{
		Observable: NewObservable[T](),
		value:      initialValue,
	}
}

func (o *ObservableValue[T]) Get() T {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.value
}

func (o *ObservableValue[T]) Set(value T) {
	o.mu.Lock()
	o.value = value
	o.mu.Unlock()
	o.TriggerAll(value)
}
