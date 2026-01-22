package node

import (
	"errors"

	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

var (
	ErrDirectoryAlreadyLoaded = errors.New("this directory is already loaded")
	ErrWrongDirectory         = errors.New("wrong directory")
)

type DirectoryNode interface {
	Node
	Path() directory.Path
	Directory() *directory.Directory
	Load(*directory.Directory) error
}

type directoryNodeImpl struct {
	baseNode
	dir     *directory.Directory
	loaded  bool
	dirPath directory.Path
}

var (
	_ Node          = (*directoryNodeImpl)(nil)
	_ DirectoryNode = (*directoryNodeImpl)(nil)
)

func NewDirectoryNode(path directory.Path, opts ...Option) DirectoryNode {
	var icon = theme.FolderIcon()
	if path == directory.RootPath {
		icon = theme.StorageIcon()
	}

	b := baseNode{
		id:          path.String(),
		displayName: path.DirectoryName(),
		nodeType:    FolderNodeType,
		icon:        icon,
	}

	for _, opt := range opts {
		opt(&b)
	}

	return &directoryNodeImpl{
		baseNode: b,
		dir:      nil,
		dirPath:  path,
	}
}

func (n *directoryNodeImpl) Path() directory.Path {
	return n.dirPath
}

func (n *directoryNodeImpl) Directory() *directory.Directory {
	if n.dir == nil {
		panic("programming error, directory not loaded")
	}
	return n.dir
}

func (n *directoryNodeImpl) Load(dir *directory.Directory) error {
	if n.loaded {
		return ErrDirectoryAlreadyLoaded
	}
	if n.dirPath != dir.Path() {
		return ErrWrongDirectory
	}

	n.loaded = true
	n.dir = dir
	return nil
}
