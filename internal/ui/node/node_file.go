package node

import "github.com/thomas-marquis/s3-box/internal/domain/directory"

type FileNode interface {
	Node
	File() *directory.File
}

type fileNodeImpl struct {
	baseNode
	file *directory.File
}

var (
	_ Node     = (*fileNodeImpl)(nil)
	_ FileNode = (*fileNodeImpl)(nil)
)

func NewFileNode(file *directory.File, opts ...Option) FileNode {
	b := baseNode{
		id:          file.FullPath(),
		nodeType:    FileNodeType,
		displayName: file.Name().String(),
		icon:        nil,
	}

	for _, opt := range opts {
		opt(&b)
	}

	return &fileNodeImpl{
		b,
		file,
	}
}

func (n *fileNodeImpl) File() *directory.File {
	return n.file
}
