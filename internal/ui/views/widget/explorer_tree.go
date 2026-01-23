package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
)

type ExplorerTree struct {
	widget.BaseWidget
	appCtx      appcontext.AppContext
	onFileClick func(file *directory.File)
	onDirClick  func(directory *directory.Directory)
}

func NewExplorerTree(
	appCtx appcontext.AppContext,
	onDirClick func(directory *directory.Directory),
	onFileClick func(file *directory.File),
) *ExplorerTree {
	w := &ExplorerTree{
		appCtx:      appCtx,
		onDirClick:  onDirClick,
		onFileClick: onFileClick,
	}

	w.ExtendBaseWidget(w)

	return w
}

func (w *ExplorerTree) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)
	vm := w.appCtx.ExplorerViewModel()

	treeData := vm.Tree()

	tree := widget.NewTreeWithData(
		treeData,
		func(branch bool) fyne.CanvasObject {
			displayLabel := widget.NewLabel("")
			icon := widget.NewIcon(theme.FolderIcon())
			icon.Hide()
			loading := widget.NewIcon(theme.ViewRefreshIcon())
			loading.Hide()
			return container.NewHBox(icon, displayLabel, loading)
		},
		func(i binding.DataItem, branch bool, o fyne.CanvasObject) {
			nodeItem, err := i.(binding.Item[node.Node]).Get()
			if err != nil {
				panic(fmt.Errorf("unexpected type %T: %w", nodeItem, err))
			}

			c, _ := o.(*fyne.Container)
			icon := c.Objects[0].(*widget.Icon)
			displayLabel := c.Objects[1].(*widget.Label)
			loading := c.Objects[2].(*widget.Icon)

			displayLabel.SetText(nodeItem.DisplayName())

			if nodeItem.Icon() != nil {
				icon.SetResource(nodeItem.Icon())
				icon.Show()
			} else {
				icon.Hide()
			}

			if dirNode, ok := nodeItem.(node.DirectoryNode); ok {
				if dirNode.Directory().IsLoading() {
					loading.Show()
				} else {
					loading.Hide()
				}
			}
		},
	)

	w.reopenOpenedDirectories(tree)

	tree.OnBranchOpened = w.makeOnBranchCallback(true, treeData)
	tree.OnBranchClosed = w.makeOnBranchCallback(false, treeData)

	tree.OnSelected = func(uid widget.TreeNodeID) {
		nodeItem, err := treeData.GetValue(uid)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error getting value: %v", err), w.appCtx.Window())
			return
		}

		switch nodeItem.NodeType() {
		case node.FolderNodeType:
			dirNode := nodeItem.(node.DirectoryNode)
			if !dirNode.Directory().IsLoaded() && !dirNode.Directory().IsLoading() {
				if err := vm.LoadDirectory(dirNode); err != nil {
					dialog.ShowError(err, w.appCtx.Window())
					return
				}
				tree.OpenBranch(uid)
			}
			dir := dirNode.Directory()
			w.onDirClick(dir)

		case node.FileNodeType:
			file := (nodeItem.(node.FileNode)).File()
			w.onFileClick(file)
		}
	}

	return widget.NewSimpleRenderer(tree)
}

func (w *ExplorerTree) reopenOpenedDirectories(tree *widget.Tree) {
	vm := w.appCtx.ExplorerViewModel()

	_, treeContent, err := vm.Tree().Get()
	if err != nil {
		return
	}

	for key, n := range treeContent {
		if n.NodeType() == node.FolderNodeType {
			dirNode := n.(node.DirectoryNode)
			if dirNode.Directory().IsOpened() {
				tree.OpenBranch(key)
			}
		}
	}
}

func (w *ExplorerTree) makeOnBranchCallback(shouldOpen bool, data binding.Tree[node.Node]) func(uid widget.TreeNodeID) {
	return func(uid widget.TreeNodeID) {
		nodeItem, err := data.GetValue(uid)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error getting value: %v", err), w.appCtx.Window())
			return
		}

		if nodeItem.NodeType() != node.FolderNodeType {
			return
		}
		dirNode := nodeItem.(node.DirectoryNode)
		if shouldOpen {
			dirNode.Directory().Open()
		} else {
			dirNode.Directory().Close()
		}
	}
}
