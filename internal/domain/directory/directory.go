package directory

import (
	"fmt"
	"strings"

	"github.com/thomas-marquis/s3-box/internal/domain/connections"
)

const (
	RootDirName   = ""
	NilParentPath = Path("")
	RootPath      = Path("/")
)

type Directory struct {
	connectionID   connections.ConnectionID
	path           Path
	name           string
	parentPath     Path
	subDirectories []Path
	files          []*File
}

// Directory creates a new S3 directory
// returns an error when the directory name is not valid
func New(
	connectionID connections.ConnectionID,
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

// NewSubDirectory reference a new subdirectory in the current one
// returns an error when the subdirectory already exists
func (d *Directory) NewSubDirectory(name string) (*Directory, error) {
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
	return newDir, nil
}

// NewFile creates a new file in the current directory
// returns an error when the file name is not valid or if the file already exists
func (d *Directory) NewFile(name string) (*File, error) {
	file, err := NewFile(name, d)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	for _, f := range d.files {
		if f.Is(file) {
			return nil, fmt.Errorf("file %s already exists in directory %s", name, d.path)
		}
	}
	d.files = append(d.files, file)
	return file, nil
}

func (d *Directory) RemoveFile(name FileName) error {
	for i, file := range d.files {
		if file.Name() == name {
			d.files = append(d.files[:i], d.files[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (d *Directory) RemoveSubDirectory(name string) error {
	path := d.parentPath.NewSubPath(name)
	for i, subDir := range d.subDirectories {
		if subDir == path {
			d.subDirectories = append(d.subDirectories[:i], d.subDirectories[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
