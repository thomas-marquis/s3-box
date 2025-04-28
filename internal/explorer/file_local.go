package explorer

import (
	"os"
	"path/filepath"
)

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
