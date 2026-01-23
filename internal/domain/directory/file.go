package directory

import (
	"errors"
	"strings"
	"time"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

// FileName is a value object representing a file name.
type FileName string

func NewFileName(name string) (FileName, error) {
	if name == "" {
		return "", errors.New("fileObj name is empty")
	}
	if name == "/" || strings.Contains(name, "/") {
		return "", errors.New("fileObj name is not valid: should not be '/' or contain '/'")
	}

	return FileName(name), nil
}

func (name FileName) String() string {
	return string(name)
}

type File struct {
	name          FileName
	directoryPath Path
	sizeBytes     int
	lastModified  time.Time
}

func NewFile(name string, parentPath Path, opts ...FileOption) (*File, error) {
	fileName, err := NewFileName(name)
	if err != nil {
		return nil, err
	}
	f := &File{
		name:          fileName,
		directoryPath: parentPath,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f, nil
}

func (f *File) Is(other *File) bool {
	if other == nil {
		return false
	}
	return f.name == other.name && f.directoryPath == other.directoryPath
}

func (f *File) Equal(other *File) bool {
	if other == nil {
		return false
	}
	return f.Is(other) &&
		f.sizeBytes == other.sizeBytes &&
		f.lastModified.Equal(other.lastModified)
}

// Name is the File entity's unique ID within a given Directory.
func (f *File) Name() FileName {
	return f.name
}

// DirectoryPath returns the path of the directory containing the file.
func (f *File) DirectoryPath() Path {
	return f.directoryPath
}

func (f *File) SizeBytes() int {
	return f.sizeBytes
}

func (f *File) LastModified() time.Time {
	return f.lastModified
}

// FullPath returns the full path of the file in the directory.
// FullPath is unique within a given bucket.
func (f *File) FullPath() string {
	return f.directoryPath.String() + f.name.String()
}

func (f *File) Download(connID connection_deck.ConnectionID, toPath string) ContentDownloadedEvent {
	return NewContentDownloadedEvent(connID, NewFileContent(f, FromLocalFile(toPath)))
}
