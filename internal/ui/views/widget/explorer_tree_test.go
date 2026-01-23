package widget_test

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"
	mocks_appcontext "github.com/thomas-marquis/s3-box/mocks/context"
	mocks_viewmodel "github.com/thomas-marquis/s3-box/mocks/viewmodel"
	"go.uber.org/mock/gomock"
)

func TestExplorerTree(t *testing.T) {
	fyne_test.NewApp()

	ctrl := gomock.NewController(t)
	mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
	mockExplorerVM := mocks_viewmodel.NewMockExplorerViewModel(ctrl)

	mockAppCtx.EXPECT().ExplorerViewModel().Return(mockExplorerVM).AnyTimes()

	treeData := binding.NewTree[node.Node](func(n1 node.Node, n2 node.Node) bool {
		return n1.ID() == n2.ID()
	})

	connID := connection_deck.NewConnectionID()
	root, _ := directory.New(connID, directory.RootDirName, directory.NilParentPath)
	rootNode := node.NewDirectoryNode(root)
	_ = treeData.Append("", rootNode.ID(), rootNode)

	child, _ := directory.New(connID, "child", root.Path())
	childDir := node.NewDirectoryNode(child)
	_ = treeData.Append(rootNode.ID(), childDir.ID(), childDir)

	file, _ := directory.NewFile("test.txt", root.Path())
	childFile := node.NewFileNode(file)
	_ = treeData.Append(rootNode.ID(), childFile.ID(), childFile)

	t.Run("should display explorer tree", func(t *testing.T) {
		// Given
		mockExplorerVM.EXPECT().Tree().Return(treeData).AnyTimes()

		// When
		res := widget.NewExplorerTree(mockAppCtx, func(directory *directory.Directory) {}, func(file *directory.File) {})
		c := fyne_test.NewWindow(res).Canvas()

		// Then
		fyne_test.AssertRendersToMarkup(t, "explorer_tree", c)
	})
}
