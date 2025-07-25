package node

import (
	"fyne.io/fyne/v2"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

const (
	FolderNodeType = "folder"
	FileNodeType   = "file"
)

type Node interface {
	ID() string
	NodeType() string
	DisplayName() string
	Icon() fyne.Resource
}

type FileNode interface {
	File() *directory.File
}

type DirectoryNode interface {
	Directory() *directory.Directory
}

type baseNode struct {
	id          string
	nodeType    string
	displayName string
	icon        fyne.Resource
}

func (n baseNode) ID() string {
	return n.id
}

func (n baseNode) NodeType() string {
	return n.nodeType
}

func (n baseNode) DisplayName() string {
	return n.displayName
}

func (n baseNode) Icon() fyne.Resource {
	return n.icon
}

type fileNodeImpl struct {
	baseNode
	file *directory.File
}

func NewFileNode(file *directory.File) FileNode {
	return fileNodeImpl{
		baseNode{
			id:          file.FullPath(),
			nodeType:    FileNodeType,
			displayName: file.Name().String(),
			icon:        nil,
		},
		file,
	}
}

func (n fileNodeImpl) File() *directory.File {
	return n.file
}

type directoryNodeImpl struct {
	baseNode
	dir *directory.Directory
}

func NewDirectoryNode(dir *directory.Directory) DirectoryNode {
	var icon fyne.Resource
	if dir
	return directoryNodeImpl{
		baseNode{
			id: dir.Path().String(),
			displayName: dir.Path().DirectoryName(),
			nodeType:    FolderNodeType,
			icon:        icon,
		},
		dir,,
	}
}

func (n directoryNodeImpl) Directory() *directory.Directory {
	return n.dir
}
