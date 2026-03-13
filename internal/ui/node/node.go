package node

import (
	"fyne.io/fyne/v2"
)

type Node interface {
	ID() string
	DisplayName() string
	Icon() fyne.Resource
	StatusMessage() string
	StatusTitle() string
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

func (n *baseNode) DisplayName() string {
	return n.displayName
}

func (n *baseNode) Icon() fyne.Resource {
	return n.icon
}

func (n *baseNode) StatusMessage() string {
	return ""
}

func (n *baseNode) StatusTitle() string {
	return ""
}
