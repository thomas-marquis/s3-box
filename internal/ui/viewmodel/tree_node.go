package viewmodel

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type TreeNodeType string

const (
	TreeNodeTypeFile       TreeNodeType = "file"
	TreeNodeTypeDirectory  TreeNodeType = "directory"
	TreeNodeTypeBucketRoot TreeNodeType = "root"
)

type TreeNode struct {
	ID          string
	DisplayName string
	Icon        fyne.Resource
	Type        TreeNodeType

	loaded bool
}

func NewTreeNode(id string, displayName string, nodeType TreeNodeType) *TreeNode {
	var i fyne.Resource = nil
	switch nodeType {
	case TreeNodeTypeDirectory:
		i = theme.FolderIcon()
	case TreeNodeTypeBucketRoot:
		i = theme.StorageIcon()
	}

	return &TreeNode{
		ID:          id,
		DisplayName: displayName,
		Icon:        i,
		loaded:      false,
		Type:        nodeType,
	}
}

func (n *TreeNode) SetIsLoaded() {
	n.loaded = true
	switch n.Type {
	case TreeNodeTypeDirectory:
		n.Icon = theme.FolderOpenIcon()
	}
}

func (n *TreeNode) SetIsNotLoaded() {
	n.loaded = false
	switch n.Type {
	case TreeNodeTypeDirectory:
		n.Icon = theme.FolderIcon()
	}
}

func (n *TreeNode) IsLoaded() bool {
	return n.loaded
}

