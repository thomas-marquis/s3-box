package directory

import (
	"fmt"
	"strings"
)

const (
	RootDirName   = ""
	NilParentPath = Path("")
	RootPath      = Path("/")
)

type Directory struct {
	path           Path
	name           string
	parentPath     Path
	subDirectories []Path
	files          []*File
}

// Directory creates a new S3 directory
// returns an error when the directory name is not valid
func New(name string, parentPath Path) (*Directory, error) {
	if name == RootDirName && parentPath != NilParentPath {
		return nil, fmt.Errorf("directory name is empty")
	}
	if name == "/" {
		return nil, fmt.Errorf("directory name should not be '/'")
	}
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("directory name should not contain '/'s")
	}

	return &Directory{
		name:           name,
		parentPath:     parentPath,
		path:           parentPath.NewSubPath(name),
		subDirectories: make([]Path, 0),
		files:          make([]*File, 0),
	}, nil
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
	newDir, err := New(name, d.parentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create sudirectory: %w", err)
	}

	d.subDirectories = append(d.subDirectories, newDir.path)
	return newDir, nil
}

func (d *Directory) NewFile(name string) (*File, error) {
	file, err := NewFile(name, d)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	for _, f := range d.files {
		if f.Name() == name {
			return nil, fmt.Errorf("file %s already exists in directory %s", name, d.path)
		}
	}
	d.files = append(d.files, file)
	return file, nil
}
