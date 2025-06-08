package tests_test

import (
	"testing"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/tests"
)

func Test_AreTreesEqual_ShouldReturnTrueWhenTreesAreEqual(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)
	tree1 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := tests.AreTreesEqual(tree1, tree2)

	// Then
	assert.True(t, result, "The trees should be equal")
	assert.Equal(t, "", report, "The report should be empty")
}

func Test_AreTreesEqual_ShouldReturnFalseWhenATreeNotContainsPointers(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)
	tree1 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithNonPointerFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := tests.AreTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t,
		`Error casting expected node (ID=/file.txt) as a pointer of viewmodel.TreeNode
`,
		report,
		"The report should contain the error message",
	)
}

func Test_AreTreesEqual_ShouldReturnFalseWhenTreesAreNotEqual(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)
	tree1 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "differentFile.txt").
		Build()

	// When
	result, report := tests.AreTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t,
		`Node with id /file.txt mismatch: actual: TreeNode{ID: /file.txt, DisplayName: file.txt, Type: file, loaded: false}, expected: TreeNode{ID: /file.txt, DisplayName: differentFile.txt, Type: file, loaded: false}
`,
		report, "The report should contain the correct error message",
	)
}

func Test_AreTreesEqual_ShouldReturnFalseWhenLessNodesInSecondTree(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)
	tree1 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	tree2 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		Build()

	// When
	result, report := tests.AreTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t, `The actual node with id /file.txt (TreeNode{ID: /file.txt, DisplayName: file.txt, Type: file, loaded: false}) does not exists in the expected nodes
`,
		report, "The report should contain the correct error message")
}

func Test_AreTreesEqual_ShouldReturnFalseWhenLessNodesInFirstTree(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)
	tree1 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		Build()

	tree2 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithDirNode("/someDir/", "/", "somedir").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := tests.AreTreesEqual(tree1, tree2)

	// Then
	assert.False(t, result, "The trees should not be equal")
	assert.Equal(t, `The expected node with id /file.txt (TreeNode{ID: /file.txt, DisplayName: file.txt, Type: file, loaded: false}) does not exists in the actual nodes
`,
		report, "The report should contain the correct error message")
}

func Test_AreTreesEqual_ShouldReturnTrueWhenSameTreesButDifferentOrder(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)
	tree1 := tests.NewUntypedTreeBuilder().
		WithRootNode("Bucket: MyBucket").
		WithFileNode("/file.txt", "/", "file.txt").
		WithDirNode("/someDir/", "/", "somedir").
		Build()

	tree2 := tests.NewUntypedTreeBuilder().
		WithDirNode("/someDir/", "/", "somedir").
		WithRootNode("Bucket: MyBucket").
		WithFileNode("/file.txt", "/", "file.txt").
		Build()

	// When
	result, report := tests.AreTreesEqual(tree1, tree2)

	// Then
	assert.True(t, result, "The trees should be equal")
	assert.Equal(t, "", report)
}
