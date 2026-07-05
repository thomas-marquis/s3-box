package settings

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thomas-marquis/it-happened/carrier"
	"github.com/thomas-marquis/it-happened/event"
)

var (
	ErrTimeout       = errors.New("settings timeout")
	ErrAlreadyExists = errors.New("setting already exists")
	ErrUnregistered  = errors.New("setting not registered")
)

type Registration func(*SettingsV3) error

func AString(name, defaultValue string) Registration {
	return func(s *SettingsV3) error {
		return s.register(name, StringType, defaultValue)
	}
}

func AUint64(name string, defaultValue uint64) Registration {
	return func(s *SettingsV3) error {
		return s.register(name, Uint64Type, defaultValue)
	}
}

func ADuration(name string, defaultValue time.Duration) Registration {
	return func(s *SettingsV3) error {
		return s.register(name, DurationType, defaultValue)
	}
}

type SType int

const (
	Uint64Type SType = iota
	StringType
	DurationType
)

type SettingsV3 struct {
	pendingEvents []event.Event

	registered map[string]SType
	values     map[SType]map[string]any
	isReady    bool

	observers  map[string]map[int]func(value any)
	mu         sync.RWMutex
	currObsIdx int
}

func NewSettingsV3() *SettingsV3 {
	return &SettingsV3{
		registered: make(map[string]SType),
		values:     make(map[SType]map[string]any),
	}
}

func (s *SettingsV3) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isReady
}

func (s *SettingsV3) Observe(name string, f func(value any)) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() { s.currObsIdx++ }()

	if _, ok := s.observers[name]; !ok {
		s.observers[name] = make(map[int]func(value any))
	}
	s.observers[name][s.currObsIdx] = f
	currIdx := s.currObsIdx

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if funcs, found := s.observers[name]; found {
			if _, stillHere := funcs[currIdx]; stillHere {
				delete(s.observers[name], currIdx)
			}
		}
	}
}

func (s *SettingsV3) Register(regs ...Registration) error {
	for _, reg := range regs {
		if err := reg(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsV3) Write(name string, value any) error {
	tp, err := inferType(value)
	if err != nil {
		return err
	}
	if !s.isRegistered(name, tp) {
		return errors.Join(ErrUnregistered, fmt.Errorf("writing %s", name))
	}

	s.mu.Lock()
	s.pendingEvents = append(s.pendingEvents, event.New(WriteTriggered{Name: name, Value: value}))
	s.mu.Unlock()
	return nil
}

func (s *SettingsV3) IsExists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, exists := s.registered[name]; exists {
		return true
	}
	return false
}

func (s *SettingsV3) IsExistsWithType(name string, tp SType) bool {
	if !s.IsExists(name) {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if tp != s.registered[name] {
		return false
	}
	return true
}

func (s *SettingsV3) ReadString(name string) string {
	if !s.isRegistered(name, StringType) {
		panic(errors.Join(ErrUnregistered, errors.New(name)))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.values[StringType][name]; ok {
		return val.(string)
	}
	return ""
}

func (s *SettingsV3) ReadUint64(name string) uint64 {
	if !s.isRegistered(name, Uint64Type) {
		panic(errors.Join(ErrUnregistered, errors.New(name)))
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.values[Uint64Type][name]; ok {
		return val.(uint64)
	}
	return 0
}

func (s *SettingsV3) ReadDuration(name string) time.Duration {
	if !s.isRegistered(name, DurationType) {
		panic(errors.Join(ErrUnregistered, errors.New(name)))
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.values[DurationType][name]; ok {
		return val.(time.Duration)
	}
	return 0
}

func (s *SettingsV3) Load() (event.Event, error) {
	s.isReady = false
	return event.New(LoadTriggered{}), nil
}

func (s *SettingsV3) Save() event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pendingEvents) == 0 {
		return event.New(SaveSucceeded{})
	}

	var evts []event.Event
	for _, evt := range s.pendingEvents {
		evts = append(evts, evt)
	}
	s.pendingEvents = nil
	s.isReady = false

	return carrier.NewAll(evts,
		func(evtCarrier event.Event, received []event.Event) event.Event {
			return evtCarrier.NewFollowup(SaveSucceeded{})
		},
		event.New(SaveFailed{
			Err:    ErrTimeout,
			Events: evts,
		}),
	)
}

func (s *SettingsV3) Notify(evt event.Event) error {
	switch pl := evt.Payload().(type) {
	case LoadSucceeded:
		s.mu.RLock()

		s.isReady = true

		// merge the received values with registered ones:
		for name, sType := range s.registered {
			if inSType, found := pl.Registered[name]; !found || inSType != sType {
				continue
			}
			if newVal, found := pl.Values[name]; found {
				s.values[sType][name] = newVal
			}
		}
		s.mu.RUnlock()

	case SaveFailed:
		s.mu.Lock()
		s.isReady = true
		s.pendingEvents = append(s.pendingEvents, pl.Events...)
		s.mu.Unlock()

	case SaveSucceeded:
		s.mu.Lock()
		s.isReady = true
		s.mu.Unlock()

	case RegisterSucceeded:
		s.mu.Lock()
		s.registered[pl.Name] = pl.Type
		s.mu.Unlock()

	case WriteSucceeded:
		if err := s.storeValue(pl.Name, pl.Value); err != nil {
			return err
		}
		s.mu.RLock()
		if observers, ok := s.observers[pl.Name]; ok {
			for _, observer := range observers {
				observer(pl.Value)
			}
		}
		s.mu.RUnlock()
	}

	return nil
}

func (s *SettingsV3) register(name string, tp SType, defaultValue any) error {
	if s.IsExists(name) {
		return errors.Join(ErrAlreadyExists, errors.New(name))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.values[tp]; !ok {
		s.values[tp] = make(map[string]any)
	}
	s.registered[name] = tp
	s.values[tp][name] = defaultValue
	return nil
}

func (s *SettingsV3) isRegistered(name string, tp SType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.registered[name]
	return ok && val == tp
}

func (s *SettingsV3) storeValue(name string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.registered[name]; !ok {
		return errors.Join(ErrUnregistered, errors.New(name))
	}

	tp, err := inferType(value)
	if err != nil {
		return err
	}
	s.values[tp][name] = value

	return nil
}

func inferType(value any) (SType, error) {
	switch value.(type) {
	case string:
		return StringType, nil
	case uint64:
		return Uint64Type, nil
	case time.Duration:
		return DurationType, nil
	default:
		return -1, errors.New("unsupported type")
	}
}
