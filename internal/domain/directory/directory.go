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

	name       string
	parentPath Path

	currentState State
}

// New creates a new S3 directory entity.
// An error is returned when the directory name is not valid
func New(
	connectionID connection_deck.ConnectionID,
	name string,
	parentPath Path,
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
		connectionID: connectionID,
		name:         name,
		parentPath:   parentPath,
		path:         parentPath.NewSubPath(name),
	}

	d.currentState = newNotLoadedState(d)

	return d, nil
}

func (d *Directory) IsFileExists(name FileName) bool {
	files, err := d.currentState.Files()
	if err != nil {
		return false
	}
	for _, file := range files {
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
	files, err := d.currentState.Files()
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Name() == name {
			return file, nil
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
	return d.parentPath
}

func (d *Directory) SubDirectories() ([]*Directory, error) { // TODO: to remove
	return d.currentState.SubDirectories()
}

func (d *Directory) Files() ([]*File, error) {
	return d.currentState.Files()
}

func (d *Directory) ConnectionID() connection_deck.ConnectionID {
	return d.connectionID
}

// NewSubDirectory reference a new subdirectory in the current one
// returns an error when the subdirectory already exists
func (d *Directory) NewSubDirectory(name string) (CreatedEvent, error) {
	path := d.path.NewSubPath(name)
	subDirs, err := d.currentState.SubDirectories()
	if err != nil {
		return CreatedEvent{}, fmt.Errorf("failed to get subdirectories: %w", err)
	}
	for _, subDir := range subDirs {
		if subDir.Path() == path {
			return CreatedEvent{}, fmt.Errorf("subdirectory %s already exists", path)
		}
	}
	newDir, err := New(d.connectionID, name, d.path)
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
	files, err := d.currentState.Files()
	if err != nil {
		return FileDeletedEvent{}, fmt.Errorf("failed to get files: %w", err)
	}
	for _, file := range files {
		if file.Name() == name {
			return NewFileDeletedEvent(d.connectionID, d, file), nil
		}
	}
	return FileDeletedEvent{}, ErrNotFound
}

func (d *Directory) RemoveSubDirectory(name string) (DeletedEvent, error) {
	path := d.parentPath.NewSubPath(name)
	subDirectories, err := d.currentState.SubDirectories()
	if err != nil {
		return DeletedEvent{}, fmt.Errorf("failed to get subdirectories: %w", err)
	}
	for _, sd := range subDirectories {
		if sd.Path() == path {
			return NewDeletedEvent(d, path), nil
		}
	}
	return DeletedEvent{}, ErrNotFound
}

func (d *Directory) UploadFile(localPath string, overwrite bool) (ContentUploadedEvent, error) {
	return d.currentState.UploadFile(localPath, overwrite)
}

// Notify processes of various event types and updates the state of the directory accordingly.
func (d *Directory) Notify(evt event.Event) error {
	switch evt.Type() {
	case DeletedEventType.AsSuccess():
		e := evt.(DeletedSuccessEvent)
		subDirectories, err := d.currentState.SubDirectories()
		if err != nil {
			return fmt.Errorf("failed to get subdirectories: %w", err)
		}
		for i, subDirPath := range subDirectories {
			if subDirPath.Is(e.Directory()) {
				if err := d.currentState.SetSubDirectories(append(subDirectories[:i], subDirectories[i+1:]...)); err != nil {
					return fmt.Errorf("failed to remove subdirectory: %w", err)
				}
				return nil
			}
		}

	case FileDeletedEventType.AsSuccess():
		e := evt.(FileDeletedSuccessEvent)
		files, err := d.currentState.Files()
		if err != nil {
			return fmt.Errorf("failed to get files: %w", err)
		}
		for i, file := range files {
			if file.Is(e.File()) {
				newFiles := append(files[:i], files[i+1:]...)
				if err := d.currentState.SetFiles(newFiles); err != nil {
					return fmt.Errorf("failed to remove file: %w", err)
				}
				return nil
			}
		}

	case FileCreatedEventType.AsSuccess():
		e := evt.(FileCreatedSuccessEvent)
		files, err := d.currentState.Files()
		if err != nil {
			return fmt.Errorf("failed to get files: %w", err)
		}
		if err := d.currentState.SetFiles(append(files, e.File())); err != nil {
			return fmt.Errorf("failed to add file: %w", err)
		}

	case CreatedEventType.AsSuccess():
		e := evt.(CreatedSuccessEvent)
		subDirectories, err := d.currentState.SubDirectories()
		if err != nil {
			return fmt.Errorf("failed to get subdirectories: %w", err)
		}
		if err := d.currentState.SetSubDirectories(append(subDirectories, e.Directory())); err != nil {
			return fmt.Errorf("failed to add subdirectory: %w", err)
		}

	case ContentUploadedEventType.AsSuccess():
		e := evt.(ContentUploadedSuccessEvent)
		newFile := e.File()
		files, err := d.currentState.Files()
		if err != nil {
			return fmt.Errorf("failed to get files: %w", err)
		}
		for i, file := range files {
			if file.Is(newFile) {
				// If a file with the same name has been created in the meantime, we overwrite it
				files[i] = newFile
				if err := d.currentState.SetFiles(files); err != nil {
					return fmt.Errorf("failed to update file: %w", err)
				}
				return nil
			}
		}
		if err := d.currentState.SetFiles(append(files, newFile)); err != nil {
			return fmt.Errorf("failed to add file: %w", err)
		}

	case LoadEventType.AsSuccess():
		e := evt.(LoadSuccessEvent)
		d.SetLoaded(true)
		if err := d.currentState.SetSubDirectories(e.SubDirectories()); err != nil {
			d.SetLoaded(false)
			return fmt.Errorf("failed to set subdirectories: %w", err)
		}
		if err := d.currentState.SetFiles(e.Files()); err != nil {
			d.SetLoaded(false)
			return fmt.Errorf("failed to set files: %w", err)
		}
	}

	return nil
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

	subDirectories, _ := d.currentState.SubDirectories() //nolint:errcheck
	files, _ := d.currentState.Files()                   //nolint:errcheck

	otherSubDirectories, _ := other.currentState.SubDirectories()
	otherFiles, _ := other.currentState.Files()

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
