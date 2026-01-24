package directory

type LoadedState struct {
	baseState
}

var _ State = (*LoadedState)(nil)

func newLoadedState(previous baseState) *LoadedState {
	return &LoadedState{previous.Clone()}
}

func (s *LoadedState) Type() StateType {
	return stateTypeLoaded
}

func (s *LoadedState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "already loaded")
}

func (s *LoadedState) SetLoaded(bool) {}

func (s *LoadedState) Open() {
	s.d.setState(newOpenedState(s.baseState))
}

func (s *LoadedState) Close() {}

func (s *LoadedState) SubDirectories() ([]*Directory, error) {
	return s.subDirs, nil
}

func (s *LoadedState) Files() ([]*File, error) {
	return s.files, nil
}

func (s *LoadedState) SetFiles(files []*File) error {
	s.files = files
	return nil
}

func (s *LoadedState) SetSubDirectories(subDirs []*Directory) error {
	s.subDirs = subDirs
	return nil
}

func (s *LoadedState) UploadFile(localPtah string, overwrite bool) (ContentUploadedEvent, error) {
	return s.d.uploadFile(localPtah, overwrite)
}
