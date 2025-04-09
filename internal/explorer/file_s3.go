package explorer

import (
	"strings"
	"time"
)

type S3File struct {
	name         string
	fullPath     string
	dirPath      string
	sizeBytes    int64
	lastModified time.Time
}

func NewS3File(fullPath string) *S3File {
	pathSplit := strings.Split(fullPath, "/")
	return &S3File{
		fullPath: fullPath,
		name:     pathSplit[len(pathSplit)-1],
		dirPath:  strings.Join(pathSplit[:len(pathSplit)-1], "/"),
	}
}

func (f *S3File) Path() string {
	return f.fullPath
}

func (f *S3File) Name() string {
	return f.name
}

func (f *S3File) DirPath() string {
	return f.dirPath
}

func (f *S3File) SizeBytes() int64 {
	return f.sizeBytes
}

func (f *S3File) SetSizeBytes(size int64) {
	f.sizeBytes = size
}

func (f *S3File) LastModified() time.Time {
	return f.lastModified
}

func (f *S3File) SetLastModified(t time.Time) {
	f.lastModified = t
}
