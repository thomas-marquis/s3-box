package directory

type NotLoadedState struct {
	baseState
}

var _ State = (*NotLoadedState)(nil)

func newNotLoadedState(d *Directory) *NotLoadedState {
	return &NotLoadedState{baseState{d: d}}
}

func (s *NotLoadedState) Type() StateType { return stateTypeNotLoaded }

func (s *NotLoadedState) Load() (LoadEvent, error) {
	s.d.setState(newLoadingState(s.baseState))
	return NewLoadEvent(s.d), nil
}

func (s *NotLoadedState) SetLoaded(bool) {}

func (s *NotLoadedState) Open() {}

func (s *NotLoadedState) Close() {}

func (s *NotLoadedState) SubDirectories() ([]*Directory, error) {
	return nil, ErrNotLoaded
}

func (s *NotLoadedState) Files() ([]*File, error) {
	return nil, ErrNotLoaded
}

func (s *NotLoadedState) SetFiles([]*File) error {
	return ErrNotLoaded
}

func (s *NotLoadedState) SetSubDirectories([]*Directory) error {
	return ErrNotLoaded
}
