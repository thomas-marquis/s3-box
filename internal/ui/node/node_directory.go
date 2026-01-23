package node

import (
	"fyne.io/fyne/v2/theme"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type DirectoryNode interface {
	Node
	Directory() *directory.Directory
}

type directoryNodeImpl struct {
	baseNode
	dir *directory.Directory
}

var (
	_ Node          = (*directoryNodeImpl)(nil)
	_ DirectoryNode = (*directoryNodeImpl)(nil)
)

func NewDirectoryNode(dir *directory.Directory, opts ...Option) DirectoryNode {
	var icon = theme.FolderIcon()
	path := dir.Path()
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
	}
}

func (n *directoryNodeImpl) Directory() *directory.Directory {
	if n.dir == nil {
		panic("programming error, directory not loaded")
	}
	return n.dir
}
