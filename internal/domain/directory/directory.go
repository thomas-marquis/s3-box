package directory

import (
	"errors"
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
	name         string
	parent       *Directory
	isOpen       bool

	currentState state
}

func NewRoot(connectionID connection_deck.ConnectionID) (*Directory, error) {
	return New(connectionID, RootDirName, nil)
}

// New creates a new S3 directory entity.
// An error is returned when the directory name is not valid
func New(
	connectionID connection_deck.ConnectionID,
	name string,
	parent *Directory,
) (*Directory, error) {
	if parent == nil {
		if name != RootDirName {
			return nil, errors.New("parent directory is nil")
		}
		parent = &Directory{
			path: NilParentPath,
		}
	}

	if err := validateName(name, parent.Path()); err != nil {
		return nil, err
	}

	d := &Directory{
		connectionID: connectionID,
		name:         name,
		parent:       parent,
		path:         parent.Path().NewSubPath(name),
	}

	d.currentState = newNotLoadedState(d, nil)

	return d, nil
}

func (d *Directory) IsFileExists(name FileName) bool {
	files := d.currentState.Files()
	for _, file := range files {
		if file.Name() == name {
			return true
		}
	}
	return false
}

func (d *Directory) IsRoot() bool {
	return d.parent.Path() == NilParentPath && d.path == RootPath
}

func (d *Directory) GetFileByName(name FileName) (*File, error) {
	files := d.currentState.Files()
	for _, file := range files {
		if file.Name() == name {
			return file, nil
		}
	}
	return nil, ErrNotFound
}

func (d *Directory) GetSubDirectoryByName(name string) (*Directory, error) {
	for _, subDir := range d.currentState.SubDirectories() {
		if subDir.Name() == name {
			return subDir, nil
		}
	}
	return nil, ErrNotFound
}

// Path acts as the primary and unique entity's ID.
// A directory path is unique within a given bucket.
func (d *Directory) Path() Path {
	return d.path
}

func (d *Directory) Name() string {
	return d.name
}

func (d *Directory) ParentPath() Path {
	return d.parent.Path()
}

func (d *Directory) SubDirectories() []*Directory {
	return d.currentState.SubDirectories()
}

func (d *Directory) Files() []*File {
	return d.currentState.Files()
}

func (d *Directory) ConnectionID() connection_deck.ConnectionID {
	return d.connectionID
}

// NewSubDirectory reference a new subdirectory in the current one
// returns an error when the subdirectory already exists
func (d *Directory) NewSubDirectory(name string) (CreatedEvent, error) {
	path := d.path.NewSubPath(name)
	for _, subDir := range d.currentState.SubDirectories() {
		if subDir.Path() == path {
			return CreatedEvent{}, fmt.Errorf("subdirectory %s already exists", path)
		}
	}
	newDir, err := New(d.connectionID, name, d)
	if err != nil {
		return CreatedEvent{}, fmt.Errorf("failed to create subdirectory: %w", err)
	}

	return NewCreatedEvent(d, newDir), nil
}

// NewFile creates a new fileObj in the current directory
// returns an error when the file name is not valid or if the file already exists if overwrite is false
func (d *Directory) NewFile(name string, overwrite bool, opts ...FileOption) (FileCreatedEvent, error) {
	file, err := NewFile(name, d.Path(), opts...)
	if err != nil {
		return FileCreatedEvent{}, fmt.Errorf("failed to create file: %w", err)
	}

	if !overwrite && d.IsFileExists(file.Name()) {
		return FileCreatedEvent{}, errors.Join(
			ErrAlreadyExists,
			fmt.Errorf("file %s already exists in directory %s", name, d.path))
	}

	return NewFileCreatedEvent(d.connectionID, d, file), nil
}

func (d *Directory) RemoveFile(name FileName) (FileDeletedEvent, error) {
	files := d.currentState.Files()
	for _, file := range files {
		if file.Name() == name {
			return NewFileDeletedEvent(d.connectionID, d, file), nil
		}
	}
	return FileDeletedEvent{}, ErrNotFound
}

func (d *Directory) RemoveSubDirectory(name string) (DeletedEvent, error) {
	path := d.parent.Path().NewSubPath(name)
	subDirectories := d.currentState.SubDirectories()
	for _, sd := range subDirectories {
		if sd.Path() == path {
			return NewDeletedEvent(d, path), nil
		}
	}
	return DeletedEvent{}, ErrNotFound
}

// Rename triggers an event to change the name of the directory.
func (d *Directory) Rename(newName string) (RenameEvent, error) {
	return d.currentState.Rename(newName)
}

func (d *Directory) UploadFile(localPath string, overwrite bool) (ContentUploadedEvent, error) {
	return d.currentState.UploadFile(localPath, overwrite)
}

// Notify processes of various event types and updates the state of the directory accordingly.
func (d *Directory) Notify(evt event.Event) error {
	return d.currentState.Notify(evt)
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

	subDirectories := d.currentState.SubDirectories()
	files := d.currentState.Files()

	otherSubDirectories := other.currentState.SubDirectories()
	otherFiles := other.currentState.Files()

	if len(subDirectories) != len(otherSubDirectories) {
		return false
	}

	for _, subDir := range subDirectories {
		foundInOther := false
		for _, otherSubDir := range otherSubDirectories {
			if subDir == otherSubDir {
				foundInOther = true
				break
			}
		}
		if !foundInOther {
			return false
		}
	}

	if len(files) != len(otherFiles) {
		return false
	}
	for _, file := range files {
		foundInOther := false
		for _, otherFile := range otherFiles {
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
	return d.currentState.Type() == stateTypeLoaded
}

func (d *Directory) IsResumable() bool {
	return d.currentState.Type() == stateTypeResumable
}

func (d *Directory) Load() (LoadEvent, error) {
	return d.currentState.Load()
}

func (d *Directory) IsOpened() bool {
	return d.isOpen
}

func (d *Directory) Open() {
	d.isOpen = true
}

func (d *Directory) Close() {
	d.isOpen = false
}

func (d *Directory) Resume() (event.Event, error) {
	return d.currentState.Resume()
}

// Status returns the current status of the directory.
// Could be nil if the directory hasn't any status.
func (d *Directory) Status() Status {
	return d.currentState.Status()
}

func (d *Directory) setState(state state) {
	d.currentState = state
}

func (d *Directory) uploadFile(localPath string, overwrite bool) (ContentUploadedEvent, error) {
	fileName := filepath.Base(localPath)
	newFileEvt, err := d.NewFile(fileName, overwrite)
	if err != nil {
		return ContentUploadedEvent{}, err
	}
	newFile := newFileEvt.File()

	uploadedEvt := NewContentUploadedEvent(d, NewFileContent(newFile, FromLocalFile(localPath)))

	return uploadedEvt, nil
}

func (d *Directory) updatePath(newParentPath Path) {
	d.path = newParentPath.NewSubPath(d.name)
	for _, file := range d.currentState.Files() {
		file.updateDirectoryPath(d.path)
	}

	subDirs := d.currentState.SubDirectories()
	for _, subDir := range subDirs {
		subDir.updatePath(d.path)
	}
}

func validateName(name string, parentPath Path) error {
	if name == RootDirName && parentPath != NilParentPath {
		return fmt.Errorf("directory name is empty")
	}
	if name == "/" {
		return fmt.Errorf("directory name should not be '/'")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("directory name should not contain '/'s")
	}
	return nil
}
