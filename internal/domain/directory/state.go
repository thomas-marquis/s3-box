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
	Load() (event.Event, error)
	Status() Status
	Recover(choice RecoveryChoice) (event.Event, error)
	Files() []*File
	SubDirectories() []*Directory
	UploadFile(localPath string, overwrite bool) (event.Event, error)
	Rename(newName string) (event.Event, error)
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

func (s *baseState) UploadFile(string, bool) (event.Event, error) {
	return event.Event{}, ErrNotLoaded
}

func (s *baseState) Rename(string) (event.Event, error) {
	return event.Event{}, ErrNotLoaded
}

func (s *baseState) Status() Status {
	return nil
}

func (s *baseState) Recover(RecoveryChoice) (event.Event, error) {
	return event.Event{}, ErrNotResumable
}
