package directory

import (
	"path/filepath"
)

type UploadMode int

const (
	UploadModeSkip UploadMode = iota
	UploadModeReplace
	UploadModeDuplicate
	UploadModeDefault = UploadModeSkip
)

type FsItem struct {
	Name     string
	AbsPath  string
	BasePath string
	IsDir    bool
	Children []*FsItem
}

func (f *FsItem) RelPath() string {
	return filepath.ToSlash(f.AbsPath[len(f.BasePath):])
}
