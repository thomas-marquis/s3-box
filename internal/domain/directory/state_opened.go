package directory

type OpenedState struct {
	baseState
}

var _ State = (*OpenedState)(nil)

func newOpenedState(previous baseState) *OpenedState {
	return &OpenedState{previous.Clone()}
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
	s.d.setState(&LoadedState{s.baseState.Clone()})
}

func (s *OpenedState) SubDirectories() ([]*Directory, error) {
	return s.subDirs, nil
}

func (s *OpenedState) Files() ([]*File, error) {
	return s.files, nil
}

func (s *OpenedState) SetFiles(files []*File) error {
	if len(files) == 0 {
		return nil
	}
	s.files = files
	return nil
}

func (s *OpenedState) SetSubDirectories(subDirs []*Directory) error {
	if len(subDirs) == 0 {
		return nil
	}
	s.subDirs = subDirs
	return nil
}
