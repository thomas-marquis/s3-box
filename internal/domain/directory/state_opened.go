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
	s.d.setState(&LoadedState{s.Clone()})
}

func (s *OpenedState) SubDirectories() ([]*Directory, error) {
	return s.subDirs, nil
}

func (s *OpenedState) Files() ([]*File, error) {
	return s.files, nil
}

func (s *OpenedState) SetFiles(files []*File) error {
	s.files = files
	return nil
}

func (s *OpenedState) SetSubDirectories(subDirs []*Directory) error {
	s.subDirs = subDirs
	return nil
}
func (s *OpenedState) UploadFile(localPtah string, overwrite bool) (ContentUploadedEvent, error) {
	return s.d.uploadFile(localPtah, overwrite)
}
