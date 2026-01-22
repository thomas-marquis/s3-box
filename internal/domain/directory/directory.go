package directory

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

const (
	RootDirName   = ""
	NilParentPath = Path("")
	RootPath      = Path("/")
)

type Directory struct {
	connectionID connection_deck.ConnectionID
	path         Path

	name           string
	parentPath     Path
	subDirectories []Path
	files          []*File

	currentState State
}

// New creates a new S3 directory entity.
// An error is returned when the directory name is not valid
func New(
	connectionID connection_deck.ConnectionID,
	name string,
	parentPath Path,
	opts ...Option,
) (*Directory, error) {
	if name == RootDirName && parentPath != NilParentPath {
		return nil, fmt.Errorf("directory name is empty")
	}
	if name == "/" {
		return nil, fmt.Errorf("directory name should not be '/'")
	}
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("directory name should not contain '/'s")
	}

	d := &Directory{
		connectionID:   connectionID,
		name:           name,
		parentPath:     parentPath,
		path:           parentPath.NewSubPath(name),
		subDirectories: make([]Path, 0),
		files:          make([]*File, 0),
	}

	d.currentState = &NotLoadedState{d}

	for _, opt := range opts {
		opt(d)
	}

	return d, nil
}

func (d *Directory) IsFileExists(name FileName) bool {
	for _, file := range d.files {
		if file.Name() == name {
			return true
		}
	}
	return false
}

func (d *Directory) IsRoot() bool {
	return d.parentPath == NilParentPath && d.path == RootPath
}

func (d *Directory) GetFileByName(name FileName) (*File, error) {
	for _, file := range d.files {
		if file.Name() == name {
			return file, nil
		}
	}
	return nil, ErrNotFound
}

func (d *Directory) Path() Path {
	return d.path
}

func (d *Directory) Name() string {
	return d.name
}

func (d *Directory) ParentPath() Path {
	return d.parentPath
}

func (d *Directory) SubDirectories() []Path {
	return d.subDirectories
}

func (d *Directory) Files() []*File {
	return d.files
}

func (d *Directory) ConnectionID() connection_deck.ConnectionID {
	return d.connectionID
}

// NewSubDirectory reference a new subdirectory in the current one
// returns an error when the subdirectory already exists
func (d *Directory) NewSubDirectory(name string) (CreatedEvent, error) {
	path := d.path.NewSubPath(name)
	for _, subDir := range d.subDirectories {
		if subDir == path {
			return CreatedEvent{}, fmt.Errorf("subdirectory %s already exists", path)
		}
	}
	newDir, err := New(d.connectionID, name, d.path)
	if err != nil {
		return CreatedEvent{}, fmt.Errorf("failed to create sudirectory: %w", err)
	}

	return NewCreatedEvent(d, newDir), nil
}

// NewFile creates a new fileObj in the current directory
// returns an error when the fileObj name is not valid or if the fileObj already exists
func (d *Directory) NewFile(name string, opts ...FileOption) (FileCreatedEvent, error) {
	file, err := NewFile(name, d.Path(), opts...)
	if err != nil {
		return FileCreatedEvent{}, fmt.Errorf("failed to create fileObj: %w", err)
	}
	for _, f := range d.files {
		if f.Is(file) {
			return FileCreatedEvent{}, fmt.Errorf("fileObj %s already exists in directory %s", name, d.path)
		}
	}

	return NewFileCreatedEvent(d.connectionID, file), nil
}

func (d *Directory) RemoveFile(name FileName) (FileDeletedEvent, error) {
	for _, file := range d.files {
		if file.Name() == name {
			return NewFileDeletedEvent(d.connectionID, d, file), nil
		}
	}
	return FileDeletedEvent{}, ErrNotFound
}

func (d *Directory) RemoveSubDirectory(name string) (DeletedEvent, error) {
	path := d.parentPath.NewSubPath(name)
	for i, subDirPath := range d.subDirectories {
		if subDirPath == path {
			d.subDirectories = append(d.subDirectories[:i], d.subDirectories[i+1:]...)
			return NewDeletedEvent(d, path), nil
		}
	}
	return DeletedEvent{}, ErrNotFound
}

func (d *Directory) UploadFile(localPath string) (ContentUploadedEvent, error) {
	fileName := filepath.Base(localPath)
	newFileEvt, err := d.NewFile(fileName)
	if err != nil {
		return ContentUploadedEvent{}, err
	}
	newFile := newFileEvt.File()

	uploadedEvt := NewContentUploadedEvent(d, NewFileContent(newFile, FromLocalFile(localPath)))

	return uploadedEvt, nil
}

// Notify processes of various event types and updates the state of the directory accordingly.
func (d *Directory) Notify(evt event.Event) {
	switch evt.Type() {
	case DeletedEventType.AsSuccess():
		e := evt.(DeletedSuccessEvent)
		for i, subDirPath := range d.subDirectories {
			if subDirPath.Is(e.Directory()) {
				d.subDirectories = append(d.subDirectories[:i], d.subDirectories[i+1:]...)
				return
			}
		}

	case FileDeletedEventType.AsSuccess():
		e := evt.(FileDeletedSuccessEvent)
		for i, file := range d.files {
			if file.Is(e.File()) {
				d.files = append(d.files[:i], d.files[i+1:]...)
				return
			}
		}

	case FileCreatedEventType.AsSuccess():
		e := evt.(FileCreatedSuccessEvent)
		d.files = append(d.files, e.File())

	case CreatedEventType.AsSuccess():
		e := evt.(CreatedSuccessEvent)
		d.subDirectories = append(d.subDirectories, e.Directory().Path())

	case ContentUploadedEventType.AsSuccess():
		e := evt.(ContentUploadedSuccessEvent)
		newFile := e.Content().File()
		for i, file := range d.files {
			if file.Is(newFile) {
				// If a file with the same name has been created in the meantime, we overwrite it
				d.files[i] = newFile
				return
			}
		}
		d.files = append(d.files, newFile)
	}
}

// Is checks if the current directory is equivalent to another in their identity.
func (d *Directory) Is(other *Directory) bool {
	if other == nil {
		return false
	}
	return d.path == other.path && d.connectionID == other.connectionID
}

// Equal compares the current directory by identity and value.
func (d *Directory) Equal(other *Directory) bool {
	if !d.Is(other) {
		return false
	}

	if len(d.subDirectories) != len(other.subDirectories) {
		return false
	}

	for _, subDir := range d.subDirectories {
		foundInOther := false
		for _, otherSubDir := range other.subDirectories {
			if subDir == otherSubDir {
				foundInOther = true
				break
			}
		}
		if !foundInOther {
			return false
		}
	}

	if len(d.files) != len(other.files) {
		return false
	}
	for _, file := range d.files {
		foundInOther := false
		for _, otherFile := range other.files {
			if file.Equal(otherFile) {
				foundInOther = true
				break
			}
		}
		if !foundInOther {
			return false
		}
	}

	return true
}

func (d *Directory) IsLoading() bool {
	return d.currentState.Type() == stateTypeLoading
}

func (d *Directory) IsLoaded() bool {
	return d.currentState.Type() == stateTypeLoaded || d.currentState.Type() == stateTypeOpened
}

func (d *Directory) IsOpened() bool {
	return d.currentState.Type() == stateTypeOpened
}

func (d *Directory) Load() (LoadEvent, error) {
	return d.currentState.Load()
}

func (d *Directory) SetLoaded(loaded bool) {
	d.currentState.SetLoaded(loaded)
}

func (d *Directory) Open() {
	d.currentState.Open()
}

func (d *Directory) Close() {
	d.currentState.Close()
}

func (d *Directory) setState(state State) {
	d.currentState = state
}
