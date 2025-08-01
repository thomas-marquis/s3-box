package views

import (
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
	"github.com/thomas-marquis/s3-box/internal/ui/views/widget"

	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/views/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	fyne_widget "fyne.io/fyne/v2/widget"
)

func makeNoConnectionTopBanner(ctx appcontext.AppContext) *fyne.Container {
	return container.NewVBox(
		container.NewCenter(fyne_widget.NewLabel("No connection selected, please select a connection in the settings menu")),
		container.NewCenter(fyne_widget.NewButton("Manage connections", func() {
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
	vm := ctx.ExplorerViewModel()

	content := container.NewHSplit(fyne_widget.NewLabel(""), fyne_widget.NewLabel(""))

	vm.SelectedConnection().AddListener(binding.NewDataListener(func() {
		if vm.CurrentSelectedConnection() == nil {
			noConn.Show()
			content.Hide()
		} else {
			noConn.Hide()
			content.Show()
		}
	}))

	treeItemBuilder := components.NewTreeItemBuilder()

	tree := fyne_widget.NewTreeWithData(
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
	fileDetails := widget.NewFileDetails(ctx)
	dirDetails := widget.NewDirectoryDetails(ctx)

	tree.OnSelected = func(uid fyne_widget.TreeNodeID) {
		nodeItem, err := uiutils.GetUntypedFromTreeById[node.Node](vm.Tree(), uid)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error getting value: %v", err), ctx.Window())
			return
		}

		switch nodeItem.NodeType() {
		case node.FolderNodeType:
			dirNode := nodeItem.(node.DirectoryNode)
			if !dirNode.IsLoaded() {
				if err := vm.LoadDirectory(dirNode); err != nil {
					dialog.ShowError(err, ctx.Window())
					return
				}
				tree.OpenBranch(uid)
			}
			dir := dirNode.Directory()
			dirDetails.Render(dir)
			detailsContainer.Objects = []fyne.CanvasObject{dirDetails}

		case node.FileNodeType:
			file := (nodeItem.(node.FileNode)).File()
			fileDetails.Render(file)
			detailsContainer.Objects = []fyne.CanvasObject{fileDetails}
		}
	}

	content.Leading = container.NewScroll(tree)
	content.Trailing = detailsContainer

	return container.NewBorder(
		noConn,
		nil,
		nil,
		nil,
		content,
	), nil
}
