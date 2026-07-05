package settings

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thomas-marquis/it-happened/carrier"
	"github.com/thomas-marquis/it-happened/event"
)

var (
	ErrTimeout       = errors.New("settings timeout")
	ErrAlreadyExists = errors.New("setting already exists")
	ErrUnregistered  = errors.New("setting not registered")
	ErrInvalidType   = errors.New("invalid type")
	ErrNotReady      = errors.New("not ready")
)

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

	observers  map[string]map[int]func(value any)
	mu         sync.RWMutex
	currObsIdx int

	state State
}

func NewSettingsV3() *SettingsV3 {
	return &SettingsV3{
		registered: make(map[string]SType),
		values:     make(map[SType]map[string]any),
		observers:  make(map[string]map[int]func(value any)),
		state:      IdleState{},
	}
}

// State returns the current state of the entity
func (s *SettingsV3) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
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
	if !s.canRegister() {
		return ErrNotReady
	}

	for _, reg := range regs {
		if err := reg(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsV3) Write(name string, value any) error {
	if !s.canWrite() {
		return ErrNotReady
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tp, err := inferType(value)
	if err != nil {
		return err
	}

	registeredType, exists := s.registered[name]
	if !exists {
		return errors.Join(ErrUnregistered, fmt.Errorf("writing %s", name))
	}

	if tp != registeredType {
		return errors.Join(ErrInvalidType, fmt.Errorf("writing %s with wrong type", name))
	}

	s.pendingEvents = append(s.pendingEvents, event.New(WriteTriggered{Name: name, Value: value}))
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
	if !s.canLoad() {
		return nil, ErrNotReady
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.transitionToState(LoadingState{})
	return event.New(LoadTriggered{}), nil
}

func (s *SettingsV3) Save() (event.Event, error) {
	if !s.canSave() {
		return nil, ErrNotReady
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pendingEvents) == 0 {
		return event.New(SaveSucceeded{}), nil
	}

	var evts []event.Event
	for _, evt := range s.pendingEvents {
		evts = append(evts, evt)
	}
	s.pendingEvents = nil
	s.transitionToState(SavingState{})

	return carrier.NewAll(evts,
		func(evtCarrier event.Event, received []event.Event) event.Event {
			return evtCarrier.NewFollowup(SaveSucceeded{})
		},
		event.New(SaveFailed{
			Err:    ErrTimeout,
			Events: evts,
		}),
	), nil
}

func (s *SettingsV3) Notify(evt event.Event) error {
	switch pl := evt.Payload().(type) {
	case LoadSucceeded:
		s.mu.Lock()
		s.transitionToState(IdleState{})

		// merge the received values with registered ones:
		for name, sType := range s.registered {
			if inSType, found := pl.Registered[name]; !found || inSType != sType {
				continue
			}
			if newVal, found := pl.Values[name]; found {
				// Convert int64 to time.Duration for duration types
				if sType == DurationType {
					if ns, ok := newVal.(int64); ok {
						newVal = time.Duration(ns)
					}
				}
				if _, valueMapExists := s.values[sType]; !valueMapExists {
					s.values[sType] = make(map[string]any)
				}
				s.values[sType][name] = newVal
			} else {
				// Value not found in Values map, use default
				if val, ok := s.values[sType][name]; ok {
					s.values[sType][name] = val
				}
			}
		}
		s.mu.Unlock()

	case LoadFailed:
		s.mu.Lock()
		s.transitionToState(IdleState{})
		s.mu.Unlock()

	case SaveFailed:
		s.mu.Lock()
		s.transitionToState(IdleState{})
		s.pendingEvents = append(s.pendingEvents, pl.Events...)
		s.mu.Unlock()

	case SaveSucceeded:
		s.mu.Lock()
		s.transitionToState(IdleState{})
		s.mu.Unlock()

	case WriteSucceeded:
		s.mu.Lock()
		defer s.mu.Unlock()

		// Validate the setting is registered and type matches
		registeredType, exists := s.registered[pl.Name]
		if !exists {
			return nil // Silent no-op for unregistered settings
		}

		// Validate type matches
		inferredType, err := inferType(pl.Value)
		if err != nil {
			return err
		}
		if inferredType != registeredType {
			return errors.Join(ErrInvalidType, fmt.Errorf("WriteSucceeded for %s with wrong type", pl.Name))
		}

		// Store the value
		if _, valueMapExists := s.values[registeredType]; !valueMapExists {
			s.values[registeredType] = make(map[string]any)
		}
		s.values[registeredType][pl.Name] = pl.Value

		if observers, ok := s.observers[pl.Name]; ok {
			for _, observer := range observers {
				observer(pl.Value)
			}
		}
	}

	return nil
}

func (s *SettingsV3) transitionToState(newState State) {
	s.state = newState
}

func (s *SettingsV3) register(name string, tp SType, defaultValue any) error {
	if strings.TrimSpace(name) == "" {
		return errors.Join(ErrInvalidType, errors.New("empty or whitespace setting name"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we're in a state that allows registration
	if !s.state.CanRegister() {
		return ErrNotReady
	}

	if _, exists := s.registered[name]; exists {
		return errors.Join(ErrAlreadyExists, errors.New(name))
	}

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

func (s *SettingsV3) canRegister() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.CanRegister()
}

func (s *SettingsV3) canSave() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.CanSave()
}

func (s *SettingsV3) canLoad() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.CanLoad()
}

func (s *SettingsV3) canWrite() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.CanWrite()
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
