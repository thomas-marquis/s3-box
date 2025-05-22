package views

import (
	"fmt"

	"github.com/thomas-marquis/s3-box/internal/explorer"

	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/app/navigation"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
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
			nodeItem, ok := di.(*viewmodel.TreeNode)
			if !ok {
				panic(fmt.Sprintf("unexpected type %T", di))
			}
			treeItemBuilder.Update(o, *nodeItem)
		})

	detailsContainer := container.NewVBox()
	fileDetails := components.NewFileDetails()
	dirDetails := components.NewDirDetails()

	tree.OnSelected = func(uid widget.TreeNodeID) {
		di, err := ctx.ExplorerViewModel().Tree().GetValue(uid)
		if err != nil {
			ctx.ExplorerViewModel().ErrorChan() <- fmt.Errorf("error getting value: %v", err)
			return
		}
		nodeItem, ok := di.(*viewmodel.TreeNode)
		if !ok {
			panic(fmt.Sprintf("unexpected type %T", di))
		}

		if (nodeItem.Type == viewmodel.TreeNodeTypeDirectory || nodeItem.Type == viewmodel.TreeNodeTypeBucketRoot) && !nodeItem.IsLoaded() {
			if err := ctx.ExplorerViewModel().OpenDirectory(explorer.S3DirectoryID(nodeItem.ID)); err != nil {
				ctx.ExplorerViewModel().ErrorChan() <- err
				return
			}
			tree.OpenBranch(uid)
			nodeItem.SetIsLoaded()
		}
		if nodeItem.Type == viewmodel.TreeNodeTypeDirectory || nodeItem.Type == viewmodel.TreeNodeTypeBucketRoot {
			d, err := ctx.ExplorerViewModel().GetDirByID(explorer.S3DirectoryID(nodeItem.ID))
			if err != nil {
				panic(fmt.Sprintf("error getting directory by ID (%s) in cache: %v", nodeItem.ID, err))
			}
			dirDetails.Update(ctx, d)
			detailsContainer.Objects = []fyne.CanvasObject{dirDetails.Object()}
		} else {
			f, err := ctx.ExplorerViewModel().GetFileByID(explorer.S3FileID(nodeItem.ID))
			if err != nil {
				panic(fmt.Sprintf("error getting file by ID (%s: %s) in cache: %v", nodeItem.Type, nodeItem.ID, err))
			}
			fileDetails.Update(ctx, f)
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
