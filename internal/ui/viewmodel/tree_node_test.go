package viewmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
)

func Test_TreeNode_String_ShouldReturnStringRepr(t *testing.T) {
	testCases := []struct {
		Node     *viewmodel.TreeNode
		Expected string
	}{
		{
			Node:     viewmodel.NewTreeNode("/test.txt", "test.txt", viewmodel.TreeNodeTypeFile),
			Expected: "TreeNode{ID: /test.txt, DisplayName: test.txt, Type: file, loaded: false}",
		},
		{
			Node:     viewmodel.NewTreeNode("/test/", "test", viewmodel.TreeNodeTypeDirectory),
			Expected: "TreeNode{ID: /test/, DisplayName: test, Type: directory, loaded: false}",
		},
		{
			Node:     viewmodel.NewTreeNode("/", "Bucket: MyBucket", viewmodel.TreeNodeTypeBucketRoot),
			Expected: "TreeNode{ID: /, DisplayName: Bucket: MyBucket, Type: root, loaded: false}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Node.ID, func(t *testing.T) {
			result := tc.Node.String()
			assert.Equal(t, tc.Expected, result)
		})
	}
}
