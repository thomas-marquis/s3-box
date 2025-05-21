package viewmodel_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	mocks_connection "github.com/thomas-marquis/s3-box/mocks/connection"
	mocks_explorer "github.com/thomas-marquis/s3-box/mocks/explorer"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()

type nodeRef struct {
	ID       string
	ParentID string
	TreeNode any
}

type untypedTreeBuilder struct {
	nodes     map[string]nodeRef
	isRootSet bool
}

func newUntypedTreeBuilder() *untypedTreeBuilder {
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

func areTreesEqual(actual binding.UntypedTree, expected binding.UntypedTree) (bool, string) {
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

func Test_areTreesEqual_ShouldReturnTrueWhenTreesAreEqual(t *testing.T) {
	// Given
	tree1 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := areTreesEqual(tree1, tree2)

	// Then
	assert.True(t, result, "The trees should be equal")
	assert.Equal(t, "", report, "The report should be empty")
}

func Test_areTreesEqual_ShouldReturnFalseWhenATreeNotContainsPointers(t *testing.T) {
	// Given
	tree1 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithNonPointerFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t,
		`Error casting expected node (ID=/file.txt) as a pointer of viewmodel.TreeNode
`,
		report,
		"The report should contain the error message",
	)
}

func Test_areTreesEqual_ShouldReturnFalseWhenTreesAreNotEqual(t *testing.T) {
	// Given
	tree1 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "differentFile.txt").
		Build()

	// When
	result, report := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t,
		`Node with id /file.txt mismatch: actual: TreeNode{ID: /file.txt, DisplayName: file.txt, Type: file, loaded: false}, expected: TreeNode{ID: /file.txt, DisplayName: differentFile.txt, Type: file, loaded: false}
`,
		report, "The report should contain the correct error message",
	)
}

func Test_areTreesEqual_ShouldReturnFalseWhenLessNodesInSecondTree(t *testing.T) {
	// Given
	tree1 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		Build()

	// When
	result, report := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t, `The actual node with id /file.txt (TreeNode{ID: /file.txt, DisplayName: file.txt, Type: file, loaded: false}) does not exists in the expected nodes
`,
		report, "The report should contain the correct error message")
}

func Test_areTreesEqual_ShouldReturnFalseWhenLessNodesInFirstTree(t *testing.T) {
	// Given
	tree1 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		Build()

	tree2 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := areTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t, `The expected node with id /file.txt (TreeNode{ID: /file.txt, DisplayName: file.txt, Type: file, loaded: false}) does not exists in the actual nodes
`,
		report, "The report should contain the correct error message")
}

func Test_areTreesEqual_ShouldReturnTrueWhenSameTreesButDifferentOrder(t *testing.T) {
	// Given
	tree1 := newUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithFileNode("/file.txt", "/", "file.txt").
		WithDirNode("/someDir/", "/", "somedir").
		Build()

	tree2 := newUntypedTreeBuilder().
		WithDirNode("/someDir/", "/", "somedir").
		WithRootNode("Bucket: MyBucket").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := areTreesEqual(tree1, tree2)

	// Then
	assert.True(t, result, "The trees should be equal")
	assert.Equal(t, "", report)
}

func Test_RefreshDir_ShouldRefreshDirectoryContent(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// setup viewmodel
	mockConnRepo := mocks_connection.NewMockRepository(ctrl)
	mockDirSvc := mocks_explorer.NewMockDirectoryService(ctrl)
	mockFileSvc := mocks_explorer.NewMockFileService(ctrl)
	mockSettingsVm := mocks_viewmodel.NewMockSettingsViewModel(ctrl)

	mockSettingsVm.EXPECT().
		CurrentTimeout().
		Return(time.Duration(10)).
		AnyTimes()

	// setup fake directory structure
	fakeRootDir, _ := explorer.NewS3Directory(explorer.RootDirName, explorer.NilParentID)
	fakeFile, _ := explorer.NewS3File("config.txt", fakeRootDir)
	fakeRootDir.AddSubDirectory("subdir")
	fakeRootDir.AddFile(fakeFile)

	mockDirSvc.EXPECT().
		GetRootDirectory(gomock.AssignableToTypeOf(ctxType)).
		Return(fakeRootDir, nil).
		Times(1)
	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(ctxType), gomock.Eq(explorer.RootDirID)).
		Return(fakeRootDir, nil).
		Times(1)

	fakeSubdir, _ := explorer.NewS3Directory("subdir", fakeRootDir.ID)
	fakeSubFile, _ := explorer.NewS3File("subfile.txt", fakeSubdir)
	fakeSubdir.AddFile(fakeSubFile)
	fakeSubdir.AddSubDirectory("demo")

	mockDirSvc.EXPECT().
		GetDirectoryByID(gomock.AssignableToTypeOf(ctxType), gomock.Eq(explorer.S3DirectoryID("/subdir/"))).
		Return(fakeSubdir, nil).
		Times(1)

	// setup fake connection
	fakeConn := connection.NewConnection(
		"my connection",
		"12345",
		"AZERTY",
		"MyBucket",
		connection.AsAWSConnection("eu-west-3"),
	)

	mockConnRepo.EXPECT().
		GetSelectedConnection(gomock.AssignableToTypeOf(ctxType)).
		Return(fakeConn, nil).
		AnyTimes()

	expectedTree := newUntypedTreeBuilder().
		WithLoadedRootNode("Bucket: MyBucket").
		WithLoadedFileNode("/config.txt", "/", "config.txt").
		WithLoadedDirNode("/subdir/", "/", "subdir").
		WithLoadedFileNode("/subdir/subfile.txt", "/subdir/", "subfile.txt").
		WithDirNode("/subdir/demo/", "/subdir/", "demo").
		Build()

	// When
	vm := viewmodel.NewExplorerViewModel(mockDirSvc, mockConnRepo, mockFileSvc, mockSettingsVm)
	// vm.RefreshDir(explorer.RootDirID)
	err := vm.RefreshDir(explorer.S3DirectoryID("/subdir/"))

	// Then
	assert.NoError(t, err)
	ok, report := areTreesEqual(vm.Tree(), expectedTree)
	fmt.Println(report)
	assert.True(t, ok, "The tree structure should be equal to the expected one")
}
