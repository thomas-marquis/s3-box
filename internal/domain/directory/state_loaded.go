package directory

import (
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

type loadedState struct {
	baseState
}

var _ state = (*loadedState)(nil)

func newLoadedState(previous baseState, subDirs []*Directory, files []*File) *loadedState {
	bs := previous.Clone()
	bs.subDirs = subDirs
	bs.files = files
	return &loadedState{bs}
}

func (s *loadedState) Type() StateType {
	return stateTypeLoaded
}

func (s *loadedState) Load() (LoadEvent, error) {
	return LoadEvent{}, NewError(s.d, "already loaded")
}

func (s *loadedState) SubDirectories() ([]*Directory, error) {
	return s.subDirs, nil
}

func (s *loadedState) Files() ([]*File, error) {
	return s.files, nil
}

func (s *loadedState) UploadFile(localPtah string, overwrite bool) (ContentUploadedEvent, error) {
	return s.d.uploadFile(localPtah, overwrite)
}

func (s *loadedState) Notify(evt event.Event) error {
	switch e := evt.(type) {
	case DeletedSuccessEvent:
		for i, subDirPath := range s.subDirs {
			if subDirPath.Is(e.Directory()) {
				s.subDirs = append(s.subDirs[:i], s.subDirs[i+1:]...)
				return nil
			}
		}

	case FileDeletedSuccessEvent:
		for i, file := range s.files {
			if file.Is(e.File()) {
				newFiles := append(s.files[:i], s.files[i+1:]...)
				s.files = newFiles
				return nil
			}
		}

	case FileCreatedSuccessEvent:
		s.files = append(s.files, e.File())

	case FileRenamedSuccessEvent:
		s.updateFile(e.File())

	case CreatedSuccessEvent:
		s.subDirs = append(s.subDirs, e.Directory())

	case ContentUploadedSuccessEvent:
		f := e.File()
		if !s.updateFile(f) {
			s.files = append(s.files, f)
		}

	case RenamedSuccessEvent:
		s.d.name = e.NewName()
		s.d.path = s.d.parentPath.NewSubPath(e.NewName())

	}
	return nil
}

func (s *loadedState) updateFile(f *File) bool {
	for i, file := range s.files {
		if file.Is(f) {
			s.files[i] = f
			return true
		}
	}
	return false
}
