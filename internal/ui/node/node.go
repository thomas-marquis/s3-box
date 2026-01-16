package node

import (
	"fyne.io/fyne/v2"
)

const (
	FolderNodeType = "node.folder"
	FileNodeType   = "node.file"
)

type Node interface {
	ID() string
	NodeType() string
	DisplayName() string
	Icon() fyne.Resource
}

type Option func(node *baseNode)

func WithDisplayName(displayName string) Option {
	return func(node *baseNode) {
		node.displayName = displayName
	}
}

type baseNode struct {
	id          string
	nodeType    string
	displayName string
	icon        fyne.Resource
}

func (n *baseNode) ID() string {
	return n.id
}

func (n *baseNode) NodeType() string {
	return n.nodeType
}

func (n *baseNode) DisplayName() string {
	return n.displayName
}

func (n *baseNode) Icon() fyne.Resource {
	return n.icon
}
