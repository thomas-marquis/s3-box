package views

import (
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"

	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func makeNoConnectionTopBanner(ctx appcontext.AppContext) *fyne.Container {
	return container.NewVBox(
		container.NewCenter(widget.NewLabel("No connection selected, please select a connection in the settings menu")),
		container.NewCenter(widget.NewButton("Manage connections", func() {
			ctx.Navigate(navigation.ConnectionRoute)
		})),
	)
}

// GetFileExplorerView initializes and returns the file explorer UI layout with functionality for file and directory navigation.
// It implements the navigation.View type interface.
// Returns filled the *fyne.Container and an error.
func GetFileExplorerView(ctx appcontext.AppContext) (*fyne.Container, error) {
	noConn := makeNoConnectionTopBanner(ctx)
	noConn.Hide()

	content := container.NewHSplit(widget.NewLabel(""), widget.NewLabel(""))

	ctx.ExplorerViewModel().OnDisplayNoConnectionBannerChange(func(shouldDisplay bool) {
		if shouldDisplay {
			noConn.Show()
			content.Hide()
		} else {
			noConn.Hide()
			content.Show()
		}
	})

	treeItemBuilder := components.NewTreeItemBuilder()

	tree := widget.NewTreeWithData(
		ctx.ExplorerViewModel().Tree(),
		func(branch bool) fyne.CanvasObject {
			return treeItemBuilder.NewRaw()
		},
		func(i binding.DataItem, branch bool, o fyne.CanvasObject) {
			di, _ := i.(binding.Untyped).Get()
			nodeItem, ok := di.(node.Node)
			if !ok {
				panic(fmt.Sprintf("unexpected type %T", di))
			}
			treeItemBuilder.Update(o, nodeItem)
		})

	detailsContainer := container.NewVBox()
	fileDetails := components.NewFileDetails()
	dirDetails := components.NewDirDetails()

	tree.OnSelected = func(uid widget.TreeNodeID) {
		nodeItem, err := uiutils.GetUntypedFromTreeById[node.Node](ctx.ExplorerViewModel().Tree(), uid)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error getting value: %v", err), ctx.Window())
			return
		}

		switch nodeItem.NodeType() {
		case node.FolderNodeType:
			dirNode := nodeItem.(node.DirectoryNode)
			dir := dirNode.Directory()
			if !dirNode.IsLoaded() {
				if err := ctx.ExplorerViewModel().LoadDirectory(dirNode); err != nil {
					return
				}
				tree.OpenBranch(uid)
			}
			dirDetails.Update(ctx, dir)
			detailsContainer.Objects = []fyne.CanvasObject{dirDetails.Object()}

		case node.FileNodeType:
			file := (nodeItem.(node.FileNode)).File()
			fileDetails.Update(ctx, file)
			detailsContainer.Objects = []fyne.CanvasObject{fileDetails.Object()}
		}
	}

	content.Leading = container.NewScroll(tree)
	content.Trailing = detailsContainer

	mainContainer := container.NewBorder(
		noConn,
		nil,
		nil,
		nil,
		content,
	)

	return mainContainer, nil
}
