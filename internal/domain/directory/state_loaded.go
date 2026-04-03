package directory

import (
	"errors"
	"fmt"
	"path/filepath"

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

func (s *loadedState) SubDirectories() []*Directory {
	return s.subDirs
}

func (s *loadedState) Files() []*File {
	return s.files
}

func (s *loadedState) Load() (LoadEvent, error) {
	// reload
	s.d.setState(newLoadingState(s.baseState))
	return NewLoadEvent(s.d), nil
}

func (s *loadedState) UploadFile(localPath string, overwrite bool) (FileUploadEvent, error) {
	fileName := filepath.Base(localPath)
	if !overwrite && s.d.IsFileExists(FileName(fileName)) {
		return FileUploadEvent{}, errors.Join(
			ErrAlreadyExists,
			fmt.Errorf("file %s already exists in directory %s", fileName, s.d.path))
	}

	uploadedEvt := NewFileUploadEvent(s.d, localPath)

	return uploadedEvt, nil
}

func (s *loadedState) Rename(newName string) (RenameEvent, error) {
	if s.d.name == RootDirName {
		return RenameEvent{}, errors.New("cannot rename root directory")
	}

	if err := validateName(newName, s.d.parent.Path()); err != nil {
		return RenameEvent{}, err
	}

	if newName == s.d.name {
		return RenameEvent{}, fmt.Errorf("new name must be different from current name %s", s.d.name)
	}

	if _, err := s.d.parent.GetSubDirectoryByName(newName); !errors.Is(err, ErrNotFound) {
		return RenameEvent{}, fmt.Errorf("a directory with name %s already exists in %s", newName, s.d.parent.Path())
	}

	s.d.setState(newLoadingState(s.baseState))
	return NewRenameEvent(s.d, newName), nil
}

func (s *loadedState) Notify(evt event.Event) error {
	switch e := evt.(type) {
	case DeletedSuccessEvent:
		for i, subDirPath := range s.subDirs {
			if subDirPath.Is(e.Directory) {
				s.subDirs = append(s.subDirs[:i], s.subDirs[i+1:]...)
				return nil
			}
		}

	case FileDeletedSuccessEvent:
		for i, file := range s.files {
			if file.Is(e.File) {
				newFiles := append(s.files[:i], s.files[i+1:]...)
				s.files = newFiles
				return nil
			}
		}

	case FileCreatedSuccessEvent:
		s.files = append(s.files, e.File)

	case FileRenameSuccessEvent:
		for _, f := range s.files {
			if f.Is(e.File) {
				n, err := NewFileName(e.NewName)
				if err != nil {
					return err
				}
				f.name = n
				return nil
			}
		}
		return fmt.Errorf("file %s not found in directory", e.File.Name())

	case CreatedSuccessEvent:
		s.subDirs = append(s.subDirs, e.Directory)

	case FileUploadSuccessEvent:
		f := e.File
		if !s.updateFile(f) {
			s.files = append(s.files, f)
		}
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
