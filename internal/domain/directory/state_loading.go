package directory

type LoadingState struct {
	baseState
}

var _ State = (*LoadingState)(nil)

func newLoadingState(previous baseState) *LoadingState {
	return &LoadingState{previous.Clone()}
}

func (s *LoadingState) Type() StateType {
	return stateTypeLoading
}

func (s *LoadingState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "loading is still in progress")
}

func (s *LoadingState) SetLoaded(loaded bool) {
	if loaded {
		bs := s.baseState.Clone()
		bs.files = make([]*File, 0)
		bs.subDirs = make([]*Directory, 0)
		s.d.setState(newLoadedState(bs))
	} else {
		s.d.setState(newNotLoadedState(s.d))
	}
}

func (s *LoadingState) Open() {}

func (s *LoadingState) Close() {}

func (s *LoadingState) SubDirectories() ([]*Directory, error) {
	return nil, ErrNotLoaded
}

func (s *LoadingState) Files() ([]*File, error) {
	return nil, ErrNotLoaded
}

func (s *LoadingState) SetFiles([]*File) error {
	return ErrNotLoaded
}

func (s *LoadingState) SetSubDirectories([]*Directory) error {
	return ErrNotLoaded
}
