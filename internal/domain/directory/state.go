package directory

import "github.com/thomas-marquis/s3-box/internal/domain/shared/event"

type StateType int

const (
	stateTypeNotLoaded StateType = iota
	stateTypeLoading
	stateTypeLoaded
	stateTypeError
)

type state interface {
	Type() StateType
	Load() (LoadEvent, error)
	Status() Status
	Recover(choice RecoveryChoice) (event.Event, error)
	Files() []*File
	SubDirectories() []*Directory
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

func (s *baseState) SubDirectories() []*Directory {
	return make([]*Directory, 0)
}

func (s *baseState) Files() []*File {
	return make([]*File, 0)
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

func (s *baseState) Recover(RecoveryChoice) (event.Event, error) {
	return nil, ErrNotResumable
}
