package explorer

import (
	"fmt"
	"strings"
)

var ErrObjectNotFoundInDirectory = fmt.Errorf("object not found in the directory")

const (
	RootDirName = ""
	NilParentID = S3DirectoryID("")
	RootDirID   = S3DirectoryID("/")
)

type S3Directory struct {
	ID                S3DirectoryID
	Name              string
	ParentID          S3DirectoryID
	SubDirectoriesIDs []S3DirectoryID
	Files             []*S3File
}

var (
	RootDir = &S3Directory{
		Name:              RootDirName,
		ParentID:          NilParentID,
		SubDirectoriesIDs: make([]S3DirectoryID, 0),
		Files:             make([]*S3File, 0),
	}
)

// NewS3Directory creates a new S3 directory
// returns an error when the directory name is not valid
func NewS3Directory(name string, parentID S3DirectoryID) (*S3Directory, error) {
	if name == RootDirName && parentID != NilParentID {
		return nil, fmt.Errorf("directory name is empty")
	}
	if name == "/" {
		return nil, fmt.Errorf("directory name should not be '/'")
	}
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("directory name should not contain '/'s")
	}

	return &S3Directory{
		ID:                buildID(name, parentID),
		Name:              name,
		ParentID:          parentID,
		SubDirectoriesIDs: make([]S3DirectoryID, 0),
		Files:             make([]*S3File, 0),
	}, nil
}

// AddSubDirectory reference a new subdirectory in the current one
// returns an error when the subdirectory already exists
func (d *S3Directory) AddSubDirectory(name string) error {
	ID := buildID(name, d.ID)
	for _, sdID := range d.SubDirectoriesIDs {
		if sdID == ID {
			return fmt.Errorf("sub directory %s already exists in S3 directory %s", ID.String(), d.ID.String())
		}
	}
	d.SubDirectoriesIDs = append(d.SubDirectoriesIDs, ID)
	return nil
}

// CreateFile creates a new file in the current directory
// returns an error when the file name is not valid
func (d *S3Directory) CreateFile(name string) (*S3File, error) {
	file, err := NewS3File(name, d)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// AddFile adds a file to the current directory
// returns an error when the file already exists
func (d *S3Directory) AddFile(file *S3File) error {
	for _, f := range d.Files {
		if f.ID == file.ID {
			return fmt.Errorf("file %s already exists in S3 directory %s", file.Name, d.ID.String())
		}
	}
	d.Files = append(d.Files, file)
	return nil
}

// DeleteFile removes a file from the current directory
// returns an error when the file does not exist
func (d *S3Directory) DeleteFile(fileID S3FileID) error {
	for i, f := range d.Files {
		if f.ID == fileID {
			d.Files = append(d.Files[:i], d.Files[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("file %s does not exist in S3 directory %s", fileID, d.ID.String())
}

// HasFile checks if a file belongs to the current directory
func (d *S3Directory) HasFile(fileID S3FileID) bool {
	for _, f := range d.Files {
		if f.ID == fileID {
			return true
		}
	}
	return false
}

func (d *S3Directory) CreateEmptySubDirectory(name string) (*S3Directory, error) {
	subDir, err := NewS3Directory(name, d.ID)
	if err != nil {
		return nil, err
	}
	err = d.AddSubDirectory(name)
	if err != nil {
		return nil, err
	}
	return subDir, nil
}

func (d *S3Directory) RemoveSubDirectory(subDirID S3DirectoryID) error {
	found := false
	for i, sdID := range d.SubDirectoriesIDs {
		if sdID == subDirID {
			d.SubDirectoriesIDs = append(d.SubDirectoriesIDs[:i], d.SubDirectoriesIDs[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return ErrObjectNotFoundInDirectory
	}
	return nil
}

func buildID(dirName string, parentID S3DirectoryID) S3DirectoryID {
	return S3DirectoryID(parentID.String() + dirName + "/")
}
