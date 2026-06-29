package node

import (
	"fyne.io/fyne/v2"
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
	path := dir.Path()

	b := baseNode{
		id:          path.String(),
		displayName: path.DirectoryName(),
		icon:        theme.FolderIcon(),
	}

	for _, opt := range opts {
		opt(&b)
	}

	return &directoryNodeImpl{
		baseNode: b,
		dir:      dir,
	}
}

func (n *directoryNodeImpl) Icon() fyne.Resource {
	if n.dir.Path() == directory.RootPath {
		return theme.StorageIcon()
	}
	if n.dir.IsLoaded() {
		return theme.FolderOpenIcon()
	}
	return theme.FolderIcon()
}

func (n *directoryNodeImpl) Directory() *directory.Directory {
	return n.dir
}

func (n *directoryNodeImpl) StatusMessage() string {
	status := n.dir.Status()
	if status == nil {
		return ""
	}
	return status.Message()
}

func (n *directoryNodeImpl) StatusTitle() string {
	status := n.dir.Status()
	if status == nil {
		return ""
	}
	return status.Title()
}
