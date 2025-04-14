package views

import (
	"context"
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
	"go.uber.org/zap"
)

func getCurrDirectoryOrFile(di any) (bool, *explorer.S3Directory, *explorer.S3File, error) {
	switch v := di.(type) {
	case *explorer.S3Directory:
		return true, v, nil, nil
	case *explorer.S3File:
		return false, nil, v, nil
	default:
		return false, nil, nil, fmt.Errorf("unexpected type %T", v)
	}
}

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

	// Start error handler
	go func() {
		for err := range ctx.Vm().ErrorChan() {
			if err == viewmodel.ErrNoConnectionSelected {
				noConn.Show()
				content.Hide()
			} else {
				ctx.L().Error("Error in explorer view", zap.Error(err))
			}
		}
	}()

	if err := ctx.Vm().AppendDirToTree(context.Background(), explorer.RootDirID); err != nil {
		ctx.Vm().ErrorChan() <- err
	}

	treeItemBuilder := components.NewTreeItemBuilder()

	tree := widget.NewTreeWithData(
		ctx.Vm().Tree(),
		func(branch bool) fyne.CanvasObject {
			return treeItemBuilder.NewRaw()
		},
		func(i binding.DataItem, branch bool, o fyne.CanvasObject) {
			di, _ := i.(binding.Untyped).Get()
			nodeItem, ok := di.(*viewmodel.TreeNode)
			if !ok {
				ctx.Vm().ErrorChan() <- fmt.Errorf("unexpected type %T", di)
				return
			}
			treeItemBuilder.Update(o, *nodeItem)
		})
		
	tree.OnSelected = func(uid widget.TreeNodeID) {
		di, err := ctx.Vm().Tree().GetValue(uid)
		if err != nil {
			ctx.Vm().ErrorChan() <- fmt.Errorf("Error getting value: %v\n", err)
			return
		}
		nodeItem, ok := di.(*viewmodel.TreeNode)
		if !ok {
			ctx.Vm().ErrorChan() <- fmt.Errorf("ERROR unexpected type %T\n", di)
			return
		}
		fmt.Printf("Selected: %s (ID=%s)\n", nodeItem.DisplayName, nodeItem.ID)

		if nodeItem.IsDirectory && !nodeItem.Loaded {
			if err := ctx.Vm().AppendDirToTree(context.Background(), explorer.S3DirectoryID(nodeItem.ID)); err != nil {
				ctx.Vm().ErrorChan() <- err
				return
			}
			tree.OpenBranch(uid)
			nodeItem.Loaded = true
		}
	}

	detailsContainer := container.NewVBox()
	fileDetails := components.NewFileDetails()
	dirDetails := components.NewDirDetails()

	tree.OnSelected = func(uid string) {
		item, err := ctx.Vm().Tree().GetValue(uid)
		if err != nil {
			ctx.L().Error("Error getting item", zap.Error(err))
			return
		}

		isDir, d, f, err := getCurrDirectoryOrFile(item)
		if err != nil {
			ctx.L().Error("Error getting directory or file", zap.Error(err))
			return
		}

		if isDir {
			dirDetails.Update(ctx, d)
			detailsContainer.Objects = []fyne.CanvasObject{dirDetails.Object()}
		} else {
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
