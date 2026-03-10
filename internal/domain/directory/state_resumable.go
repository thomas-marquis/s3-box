package directory

import "errors"

type ResumableState struct {
	baseState
}

var _ State = (*ResumableState)(nil)

func newResumableState(previous baseState) *ResumableState {
	return &ResumableState{previous.Clone()}
}

func (s *ResumableState) Type() StateType {
	return stateResumable
}

func (s *ResumableState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "this directory is already loaded")
}

func (s *ResumableState) SetLoaded(bool) {}

func (s *ResumableState) SubDirectories() ([]*Directory, error) {
	return s.subDirs, nil
}

func (s *ResumableState) Files() ([]*File, error) {
	return s.files, nil
}

func (s *ResumableState) SetFiles(files []*File) error {
	s.files = files
	return nil
}

func (s *ResumableState) SetSubDirectories(subDirs []*Directory) error {
	s.subDirs = subDirs
	return nil
}

func (s *ResumableState) UploadFile(localPtah string, overwrite bool) (ContentUploadedEvent, error) {
	return ContentUploadedEvent{}, errors.New("you can't upload files to a resumable directory")
}
