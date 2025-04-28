package explorer

import (
	"errors"
	"strings"
	"time"
)

type S3FileID string

func (id S3FileID) String() string {
	return string(id)
}

func (id S3FileID) ToName() string {
	dirPathStriped := strings.TrimSuffix(id.String(), "/")
	dirPathSplit := strings.Split(dirPathStriped, "/")
	dirName := dirPathSplit[len(dirPathSplit)-1]
	return dirName
}

type S3File struct {
	ID           S3FileID
	DirectoryID  S3DirectoryID
	Name         string
	SizeBytes    int64
	LastModified time.Time
}

func NewS3File(name string, dir *S3Directory) (*S3File, error) {
	if name == "" {
		return nil, errors.New("file name is empty")
	}
	if name == "/" {
		return nil, errors.New("file name is not valid")
	}
	
	return &S3File{
		ID:           makeFileID(name, dir),
		Name:         name,
		DirectoryID:  dir.ID,
	}, nil
}

// TODO: delete
// TODO: move ???
// TODO: copy -> S3File
// TODO: download -> LocalFile

func makeFileID(name string, dir *S3Directory) S3FileID {
	return S3FileID(dir.ID.String() + "/" + name)
}
