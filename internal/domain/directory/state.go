package directory

type StateType int

const (
	stateTypeNotLoaded StateType = iota
	stateTypeLoading
	stateTypeLoaded
	stateTypeOpened
)

type State interface {
	Type() StateType
	Load() (LoadEvent, error)
	SetLoaded(bool)
	Open()
	Close()
}

var (
	_ State = (*NotLoadedState)(nil)
	_ State = (*LoadingState)(nil)
	_ State = (*LoadedState)(nil)
	_ State = (*OpenedState)(nil)
)

type NotLoadedState struct {
	d *Directory
}

func (s *NotLoadedState) Type() StateType { return stateTypeNotLoaded }

func (s *NotLoadedState) Load() (LoadEvent, error) {
	s.d.setState(&LoadingState{d: s.d})
	return NewLoadEvent(s.d), nil
}

func (s *NotLoadedState) SetLoaded(bool) {}

func (s *NotLoadedState) Open() {
	return
}

func (s *NotLoadedState) Close() {
	return
}

type LoadingState struct {
	d *Directory
}

func (s *LoadingState) Type() StateType {
	return stateTypeLoading
}

func (s *LoadingState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "loading is still in progress")
}

func (s *LoadingState) SetLoaded(loaded bool) {
	if loaded {
		s.d.setState(&LoadedState{d: s.d})
	} else {
		s.d.setState(&NotLoadedState{d: s.d})
	}
}

func (s *LoadingState) Open() {}

func (s *LoadingState) Close() {}

type LoadedState struct {
	d *Directory
}

func (s *LoadedState) Type() StateType {
	return stateTypeLoaded
}

func (s *LoadedState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "already loaded")
}

func (s *LoadedState) SetLoaded(bool) {}

func (s *LoadedState) Open() {
	s.d.setState(&OpenedState{s.d})
}

func (s *LoadedState) Close() {}

type OpenedState struct {
	d *Directory
}

func (s *OpenedState) Type() StateType {
	return stateTypeOpened
}

func (s *OpenedState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "already loaded and opened")
}

func (s *OpenedState) SetLoaded(bool) {}

func (s *OpenedState) Open() {}

func (s *OpenedState) Close() {
	s.d.setState(&LoadedState{s.d})
}
