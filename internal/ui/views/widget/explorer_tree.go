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
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
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
	vm := w.appCtx.ExplorerViewModel()

	tree := widget.NewTreeWithData(
		vm.Tree(),
		func(branch bool) fyne.CanvasObject {
			displayLabel := widget.NewLabel("-")
			icon := widget.NewIcon(theme.FolderIcon())
			icon.Hide()
			return container.NewHBox(icon, displayLabel)
		},
		func(i binding.DataItem, branch bool, o fyne.CanvasObject) {
			di, _ := i.(binding.Untyped).Get()
			nodeItem, ok := di.(node.Node)
			if !ok {
				panic(fmt.Sprintf("unexpected type %T", di))
			}

			c, _ := o.(*fyne.Container)
			icon := c.Objects[0].(*widget.Icon)
			displayLabel := c.Objects[1].(*widget.Label)

			displayLabel.SetText(nodeItem.DisplayName())

			if nodeItem.Icon != nil {
				icon.SetResource(nodeItem.Icon())
				icon.Show()
			} else {
				icon.Hide()
			}
		},
	)

	tree.OnSelected = func(uid widget.TreeNodeID) {
		nodeItem, err := uiutils.GetUntypedFromTreeById[node.Node](vm.Tree(), uid)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error getting value: %v", err), w.appCtx.Window())
			return
		}

		switch nodeItem.NodeType() {
		case node.FolderNodeType:
			dirNode := nodeItem.(node.DirectoryNode)
			if !dirNode.IsLoaded() {
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
