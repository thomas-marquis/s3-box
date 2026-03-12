package directory

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

type StateType int

const (
	stateTypeNotLoaded StateType = iota
	stateTypeLoading
	stateTypeLoaded
	stateTypeResumable
)

type state interface {
	Type() StateType
	Load() (LoadEvent, error)
	Status() Status
	Resume() (event.Event, error)
	Files() ([]*File, error)
	SubDirectories() ([]*Directory, error)
	UploadFile(localPath string, overwrite bool) (ContentUploadedEvent, error)
	Rename(newName string) (RenameEvent, error)
	Notify(event.Event) error
}

type baseState struct {
	d       *Directory
	files   []*File
	subDirs []*Directory
}

func (s *baseState) Clone() baseState {
	return baseState{d: s.d, files: s.files, subDirs: s.subDirs}
}

func (s *baseState) SubDirectories() ([]*Directory, error) {
	return s.subDirs, nil
}

func (s *baseState) Files() ([]*File, error) {
	return s.files, nil
}

func (s *baseState) UploadFile(string, bool) (ContentUploadedEvent, error) {
	return ContentUploadedEvent{}, ErrNotLoaded
}

func (s *baseState) Rename(string) (RenameEvent, error) {
	return RenameEvent{}, ErrNotLoaded
}

func (s *baseState) Status() Status {
	return nil
}

func (s *baseState) Resume() (event.Event, error) {
	return nil, ErrNotResumable
}
