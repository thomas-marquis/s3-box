package tests

import (
	"fmt"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
)

type nodeRef struct {
	ID       string
	ParentID string
	TreeNode any
}

type untypedTreeBuilder struct {
	nodes     map[string]nodeRef
	isRootSet bool
}

func NewUntypedTreeBuilder() *untypedTreeBuilder {
	return &untypedTreeBuilder{
		nodes: make(map[string]nodeRef),
	}
}

func (b *untypedTreeBuilder) WithDirNode(ID, parentID string, displayName string) *untypedTreeBuilder {
	b.nodes[ID] = nodeRef{ID, parentID, viewmodel.NewTreeNode(ID, displayName, viewmodel.TreeNodeTypeDirectory)}
	return b
}

func (b *untypedTreeBuilder) WithLoadedDirNode(ID, parentID string, displayName string) *untypedTreeBuilder {
	node := viewmodel.NewTreeNode(ID, displayName, viewmodel.TreeNodeTypeDirectory)
	node.SetIsLoaded()
	b.nodes[ID] = nodeRef{ID, parentID, node}
	return b
}

func (b *untypedTreeBuilder) WithFileNode(ID, parentID string, displayName string) *untypedTreeBuilder {
	b.nodes[ID] = nodeRef{ID, parentID, viewmodel.NewTreeNode(ID, displayName, viewmodel.TreeNodeTypeFile)}
	return b
}

func (b *untypedTreeBuilder) WithLoadedFileNode(ID, parentID string, displayName string) *untypedTreeBuilder {
	node := viewmodel.NewTreeNode(ID, displayName, viewmodel.TreeNodeTypeFile)
	node.SetIsLoaded()
	b.nodes[ID] = nodeRef{ID, parentID, node}
	return b
}

func (b *untypedTreeBuilder) WithRootNode(displayName string) *untypedTreeBuilder {
	if b.isRootSet {
		panic("Only one root node can be set")
	}
	b.nodes[explorer.RootDirID.String()] = nodeRef{explorer.RootDirID.String(), "", viewmodel.NewTreeNode(explorer.RootDirID.String(), displayName, viewmodel.TreeNodeTypeBucketRoot)}
	b.isRootSet = true
	return b
}

func (b *untypedTreeBuilder) WithLoadedRootNode(displayName string) *untypedTreeBuilder {
	if b.isRootSet {
		panic("Only one root node can be set")
	}
	node := viewmodel.NewTreeNode(explorer.RootDirID.String(), displayName, viewmodel.TreeNodeTypeBucketRoot)
	node.SetIsLoaded()
	b.nodes[explorer.RootDirID.String()] = nodeRef{explorer.RootDirID.String(), "", node}
	b.isRootSet = true
	return b
}

func (b *untypedTreeBuilder) WithNonPointerFileNode(ID, parentID string, displayName string) *untypedTreeBuilder {
	b.nodes[ID] = nodeRef{ID, parentID, *viewmodel.NewTreeNode(ID, displayName, viewmodel.TreeNodeTypeFile)}
	return b
}

func (b *untypedTreeBuilder) Build() binding.UntypedTree {
	if !b.isRootSet {
		panic("Root node must be set before building the tree")
	}
	t := binding.NewUntypedTree()
	for id, node := range b.nodes {
		t.Append(node.ParentID, id, node.TreeNode)
	}
	return t
}

func AreTreesEqual(actual binding.UntypedTree, expected binding.UntypedTree) (bool, string) {
	report := ""
	compare := func(a, b binding.UntypedTree, aLabel, bLabel string) bool {
		res := true
		_, aTreeContent, _ := a.Get()
		for i := range aTreeContent {
			val, _ := a.GetValue(i)
			aNode, aOk := val.(*viewmodel.TreeNode)
			if !aOk {
				report = fmt.Sprintf("%sError casting %s node (ID=%s; Value=%v) as a pointer of viewmodel.TreeNode\n", report, aLabel, i, val)
				res = false
				continue
			}
			val, err := b.GetValue(i)
			bNode, bOk := val.(*viewmodel.TreeNode)
			if val == nil || err != nil {
				report = fmt.Sprintf("%sThe %s node with id %s (%s) does not exists in the %s nodes\n", report, aLabel, i, aNode, bLabel)
				res = false
			} else if !bOk {
				report = fmt.Sprintf("%sError casting %s node (ID=%s) as a pointer of viewmodel.TreeNode\n", report, bLabel, i)
				res = false
			} else if *aNode != *bNode {
				report = fmt.Sprintf("%sNode with id %s mismatch: %s: %s, %s: %s\n", report, i, aLabel, aNode, bLabel, bNode)
				res = false
			}
		}
		return res
	}

	return compare(actual, expected, "actual", "expected") && compare(expected, actual, "expected", "actual"), report
}
