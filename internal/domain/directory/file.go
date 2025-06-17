package directory

import (
	"errors"
	"strings"
	"time"
)

type FileName string

func NewFileName(name string) (FileName, error) {
	if name == "" {
		return "", errors.New("file name is empty")
	}
	if name == "/" || strings.Contains(name, "/") {
		return "", errors.New("file name is not valid: should not be '/' or contain '/'")
	}

	return FileName(name), nil
}

func (name FileName) String() string {
	return string(name)
}

type File struct {
	name          FileName
	directoryPath Path
	sizeBytes     int64
	lastModified  time.Time
}

func NewFile(name string, dir *Directory, opts ...FileOption) (*File, error) {
	fileName, err := NewFileName(name)
	if err != nil {
		return nil, err
	}
	f := &File{
		name:          fileName,
		directoryPath: dir.Path(),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f, nil
}

func (f *File) Is(other *File) bool {
	if f == nil || other == nil {
		return false
	}
	return f.name == other.name && f.directoryPath == other.directoryPath
}

func (f *File) Name() FileName {
	return f.name
}

func (f *File) DirectoryPath() Path {
	return f.directoryPath
}

func (f *File) SizeBytes() int64 {
	return f.sizeBytes
}

func (f *File) LastModified() time.Time {
	return f.lastModified
}

func (f *File) SetSizeBytes(size int64) {
	f.sizeBytes = size
}

func (f *File) FullPath() string {
	return f.directoryPath.String() + f.name.String()
}
