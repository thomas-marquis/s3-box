package directory

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/thomas-marquis/it-happened/event"
)

type loadedState struct {
	baseState
}

var _ state = (*loadedState)(nil)

func newLoadedState(previous baseState, subDirs []*Directory, files []*File) *loadedState {
	bs := previous.Clone()
	if subDirs == nil {
		bs.subDirs = []*Directory{}
	}
	if files == nil {
		bs.files = []*File{}
	}
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

func (s *loadedState) Load() (event.Event, error) {
	// reload
	s.d.setState(newLoadingState(s.baseState))
	return event.New(LoadTriggered{Directory: s.d}), nil
}

func (s *loadedState) UploadFile(localPath string, overwrite bool) (event.Event, error) {
	fileName := filepath.Base(localPath)

	if !overwrite && s.d.IsFileExists(FileName(fileName)) {
		return nil, errors.Join(
			ErrAlreadyExists,
			fmt.Errorf("file %s already exists in directory %s", fileName, s.d.path))
	}

	uploadedEvt := event.New(UploadFileTriggered{
		Directory: s.d,
		SrcPath:   localPath,
	})

	return uploadedEvt, nil
}

func (s *loadedState) Rename(newName string) (event.Event, error) {
	if s.d.name == RootDirName {
		return nil, errors.New("cannot rename root directory")
	}

	if err := validateName(newName, s.d.parent.Path()); err != nil {
		return nil, err
	}

	if newName == s.d.name {
		return nil, fmt.Errorf("new name must be different from current name %s", s.d.name)
	}

	if _, err := s.d.parent.GetSubDirectoryByName(newName); !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("a directory with name %s already exists in %s", newName, s.d.parent.Path())
	}

	s.d.setState(newLoadingState(s.baseState))
	return event.New(RenameTriggered{
		Directory: s.d,
		NewName:   newName,
	}), nil
}

func (s *loadedState) Preview() (*Preview, error) {
	return newPreview(s.d, s.d), nil
}

func (s *loadedState) Notify(evt event.Event) error {
	switch pl := evt.Payload().(type) {
	case DeleteSucceeded:
		for i, subDirPath := range s.subDirs {
			if subDirPath.Is(pl.Directory) {
				s.subDirs = append(s.subDirs[:i], s.subDirs[i+1:]...)
				return nil
			}
		}

	case DeleteFileSucceeded:
		for i, file := range s.files {
			if file.Is(pl.File) {
				newFiles := append(s.files[:i], s.files[i+1:]...)
				s.files = newFiles
				return nil
			}
		}

	case CreateFileSucceeded:
		s.files = append(s.files, pl.File)

	case RenameFileSucceeded:
		for _, f := range s.files {
			if f.Is(pl.File) {
				n, err := NewFileName(pl.NewName)
				if err != nil {
					return err
				}
				f.name = n
				return nil
			}
		}
		return fmt.Errorf("file %s not found in directory", pl.File.Name())

	case CreateSucceeded:
		pl.Directory.setState(newLoadedState(baseState{d: pl.Directory}, nil, nil))
		s.subDirs = append(s.subDirs, pl.Directory)

	case UploadFileSucceeded:
		f := pl.File
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
