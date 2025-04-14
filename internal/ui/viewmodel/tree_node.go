package viewmodel

type TreeNode struct {
	ID string
	IsDirectory bool
	DisplayName string
	Loaded bool
}

func NewTreeNode(id string, displayName string, isDirectory bool) *TreeNode {
	return &TreeNode{
		ID: id,
		IsDirectory: isDirectory,
		DisplayName: displayName,
		Loaded: false,
	}
}