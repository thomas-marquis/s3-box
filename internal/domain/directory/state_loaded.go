package directory

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/thomas-marquis/it-happened/event"
)

type loadedState struct {
	baseState
}

var _ state = (*loadedState)(nil)

func newLoadedState(previous baseState, subDirs map[Path]*Directory, files map[FileName]*File) *loadedState {
	bs := previous.Clone()
	if subDirs == nil {
		bs.subDirs = make(map[Path]*Directory)
	} else {
		bs.subDirs = subDirs
	}
	if files == nil {
		bs.files = make(map[FileName]*File)
	} else {
		bs.files = files
	}
	return &loadedState{baseState: bs}
}

func (s *loadedState) Type() StateType {
	return stateTypeLoaded
}

func (s *loadedState) SubDirectories() (dirs []*Directory) {
	keys := slices.Collect(maps.Keys(s.subDirs))
	slices.Sort(keys)
	for _, path := range keys {
		dirs = append(dirs, s.subDirs[path])
	}
	return
}

func (s *loadedState) Files() (files []*File) {
	keys := slices.Collect(maps.Keys(s.files))
	slices.Sort(keys)
	for _, name := range keys {
		files = append(files, s.files[name])
	}
	return
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
		if _, found := s.subDirs[pl.Directory.Path()]; found {
			delete(s.subDirs, pl.Directory.Path())
			return nil
		}

	case DeleteFileSucceeded:
		delete(s.files, pl.File.Name())

	case CreateFileSucceeded:
		s.files[pl.File.Name()] = pl.File

	case RenameFileSucceeded:
		if f, found := s.files[pl.File.Name()]; found {
			n, err := NewFileName(pl.NewName)
			if err != nil {
				return err
			}
			f.name = n
			return nil
		}

		return fmt.Errorf("file %s not found in directory", pl.File.Name())

	case CreateSucceeded:
		pl.Directory.setState(newLoadedState(baseState{d: pl.Directory}, nil, nil))
		s.subDirs[pl.Directory.Path()] = pl.Directory

	case UploadFileSucceeded:
		f := pl.File
		if !s.updateFile(f) {
			s.files[f.Name()] = f
		}
	}
	return nil
}

func (s *loadedState) updateFile(f *File) bool {
	if _, found := s.files[f.Name()]; found {
		s.files[f.Name()] = f
		return true
	}

	return false
}
