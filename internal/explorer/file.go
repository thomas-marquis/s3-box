package explorer

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type S3File struct {
	name         string
	fullPath     string
	dirPath      string
	sizeBytes    int64
	lastModified time.Time
	parentDir    *S3Directory
}

func NewS3File(fullPath string, parentDir *S3Directory) *S3File {
	pathSplit := strings.Split(fullPath, "/")
	return &S3File{
		fullPath:  fullPath,
		name:      pathSplit[len(pathSplit)-1],
		dirPath:   strings.Join(pathSplit[:len(pathSplit)-1], "/"),
		parentDir: parentDir,
	}
}

func (f *S3File) Path() string {
	return f.fullPath
}

func (f *S3File) Name() string {
	return f.name
}

func (f *S3File) ParentDir() *S3Directory {
	return f.parentDir
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
