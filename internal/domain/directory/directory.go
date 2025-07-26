package directory

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

const (
	RootDirName   = ""
	NilParentPath = Path("")
	RootPath      = Path("/")
)

type Directory struct {
	connectionID   connection_deck.ConnectionID
	path           Path
	name           string
	parentPath     Path
	subDirectories []Path
	files          []*File
}

// New creates a new S3 directory entity.
// An error is returned when the directory name is not valid
func New(
	connectionID connection_deck.ConnectionID,
	name string,
	parentPath Path,
	opts ...DirectoryOption,
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

func (d *Directory) GetFile(name FileName) (*File, error) { // TODO: rename to FileWithName
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
func (d *Directory) NewSubDirectory(name string) (DirectoryEvent, error) {
	path := d.parentPath.NewSubPath(name)
	for _, subDir := range d.subDirectories {
		if subDir == path {
			return nil, fmt.Errorf("subdirectory %s already exists", path)
		}
	}
	newDir, err := New(d.connectionID, name, d.parentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create sudirectory: %w", err)
	}

	d.subDirectories = append(d.subDirectories, newDir.path)

	return newDirectoryCreatedEvent(d.connectionID, newDir), nil
}

// NewFile creates a new fileObj in the current directory
// returns an error when the fileObj name is not valid or if the fileObj already exists
func (d *Directory) NewFile(name string) (FileEvent, error) {
	file, err := NewFile(name, d)
	if err != nil {
		return nil, fmt.Errorf("failed to create fileObj: %w", err)
	}
	for _, f := range d.files {
		if f.Is(file) {
			return nil, fmt.Errorf("fileObj %s already exists in directory %s", name, d.path)
		}
	}
	d.files = append(d.files, file)

	return newFileCreatedEvent(d.connectionID, file), nil
}

func (d *Directory) RemoveFile(name FileName) (FileEvent, error) {
	for i, file := range d.files {
		if file.Name() == name {
			d.files = append(d.files[:i], d.files[i+1:]...)

			return newFileDeletedEvent(d.connectionID, file), nil
		}
	}
	return fileDeletedEvent{}, ErrNotFound
}

func (d *Directory) RemoveSubDirectory(name string) (DirectoryEvent, error) {
	path := d.parentPath.NewSubPath(name)
	for i, subDirPath := range d.subDirectories {
		if subDirPath == path {
			d.subDirectories = append(d.subDirectories[:i], d.subDirectories[i+1:]...)
			return newDirectoryDeletedEvent(d.connectionID, d), nil
		}
	}
	return directoryDeletedEvent{}, ErrNotFound
}

func (d *Directory) UploadFile(localPath string) (ContentEvent, error) {
	fileName := filepath.Base(localPath)
	newFileEvt, err := d.NewFile(fileName)
	if err != nil {
		return nil, err
	}
	newFile := newFileEvt.File()

	uploadedEvt := newContentUploadedEvent(d.connectionID, NewFileContent(newFile, FromLocalFile(localPath)))
	uploadedEvt.AttachErrorCallback(func(_ error) {
		d.RemoveFile(newFile.Name())
	})

	return uploadedEvt, nil
}
