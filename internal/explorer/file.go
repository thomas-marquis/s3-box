package explorer

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RemoteFile struct {
	name         string
	fullPath     string
	dirPath      string
	sizeBytes    int64
	lastModified time.Time
	parentDir    *Directory
}

func NewRemoteFile(fullPath string, parentDir *Directory) *RemoteFile {
	pathSplit := strings.Split(fullPath, "/")
	return &RemoteFile{
		fullPath:  fullPath,
		name:      pathSplit[len(pathSplit)-1],
		dirPath:   strings.Join(pathSplit[:len(pathSplit)-1], "/"),
		parentDir: parentDir,
	}
}

func (f *RemoteFile) Path() string {
	return f.fullPath
}

func (f *RemoteFile) Name() string {
	return f.name
}

func (f *RemoteFile) ParentDir() *Directory {
	return f.parentDir
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

type LocalFile struct {
	path string
}

func NewLocalFile(path string) *LocalFile {
	return &LocalFile{path: path}
}

func (f *LocalFile) Exists() bool {
	_, err := os.Stat(f.path)
	return !os.IsNotExist(err)
}

func (f *LocalFile) ParentDirPath() string {
	return filepath.Dir(f.path)
}

func (f *LocalFile) FileName() string {
	return filepath.Base(f.path)
}

func (f *LocalFile) Path() string {
	return f.path
}
