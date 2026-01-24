package directory

type StateType int

const (
	stateTypeNotLoaded StateType = iota
	stateTypeLoading
	stateTypeLoaded
	stateTypeOpened
)

type State interface {
	Type() StateType
	Load() (LoadEvent, error)
	SetLoaded(bool)
	Open()
	Close()

	Files() ([]*File, error)
	SubDirectories() ([]*Directory, error)

	SetFiles([]*File) error
	SetSubDirectories([]*Directory) error

	UploadFile(localPath string, overwrite bool) (ContentUploadedEvent, error)
}

type baseState struct {
	d       *Directory
	files   []*File
	subDirs []*Directory
}

func (s *baseState) Clone() baseState {
	return baseState{d: s.d, files: s.files, subDirs: s.subDirs}
}

func (s *baseState) UploadFile(string, bool) (ContentUploadedEvent, error) {
	return ContentUploadedEvent{}, ErrNotLoaded
}
