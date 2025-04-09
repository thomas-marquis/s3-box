package explorer

import (
	"strings"
	"time"
)

type RemoteFile struct {
	name         string
	fullPath     string
	dirPath      string
	sizeBytes    int64
	lastModified time.Time
}

func NewRemoteFile(fullPath string) *RemoteFile {
	pathSplit := strings.Split(fullPath, "/")
	return &RemoteFile{
		fullPath: fullPath,
		name:     pathSplit[len(pathSplit)-1],
		dirPath:  strings.Join(pathSplit[:len(pathSplit)-1], "/"),
	}
}

func (f *RemoteFile) Path() string {
	return f.fullPath
}

func (f *RemoteFile) Name() string {
	return f.name
}

func (f *RemoteFile) DirPath() string {
	return f.dirPath
}

func (f *RemoteFile) SizeBytes() int64 {
	return f.sizeBytes
}

func (f *RemoteFile) SetSizeBytes(size int64) {
	f.sizeBytes = size
}

func (f *RemoteFile) LastModified() time.Time {
	return f.lastModified
}

func (f *RemoteFile) SetLastModified(t time.Time) {
	f.lastModified = t
}
