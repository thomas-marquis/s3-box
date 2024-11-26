package explorerview

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
	"github.com/thomas-marquis/s3-box/internal/ui/viewerror"
	"go.uber.org/zap"
)

func GetFileExplorerView(ctx appcontext.AppContext) (*fyne.Container, error) {
	errChan := make(chan error)

	noConn := makeNoConnectionTopBanner(ctx)
	noConn.Hide()

	content := container.NewHSplit(widget.NewLabel(""), widget.NewLabel(""))

	go func() {
		for {
			select {
			case err := <-errChan:
				if err == viewerror.ErrNoConnectionSelected {
					noConn.Show()
					content.Hide()
				} else {
					ctx.Log().Error("Error in explorer view", zap.Error(err))
				}
			case _, ok := <-ctx.ExitChan():
				if !ok {
					return
				}
			}
		}
	}()

	if err := ctx.Vm().ExpandDir(explorer.RootDir); err != nil {
		errChan <- err
	}

	treeItemBuilder := newTreeItemBuilder()

	tree := widget.NewTreeWithData(
		ctx.Vm().Tree(),
		func(branch bool) fyne.CanvasObject {
			return treeItemBuilder.NewRaw()
		},
		func(i binding.DataItem, branch bool, o fyne.CanvasObject) {
			di, _ := i.(binding.Untyped).Get()
			isDir, d, f, err := getCurrDirectoryOrFile(di)
			if err != nil {
				errChan <- err
				return
			}

			if isDir {
				err := ctx.Vm().ExpandDir(d)
				if err != nil {
					errChan <- err
					return
				}
				if d.Path() == explorer.RootDir.Path() {
					var bucket string
					if conn := ctx.Vm().SelectedConnection(); conn != nil {
						bucket = conn.BucketName
					}
					treeItemBuilder.Update(o, "Bucket: "+bucket)
				} else {
					treeItemBuilder.Update(o, d.Name+"/")
				}
			} else {
				treeItemBuilder.Update(o, f.Name())
			}
		})

	detailsContainer := container.NewVBox()
	fileDetails := newFileDetails()
	dirDetails := newDirDetails()

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

func getCurrDirectoryOrFile(di any) (bool, *explorer.Directory, *explorer.RemoteFile, error) {
	switch v := di.(type) {
	case *explorer.Directory:
		return true, v, nil, nil
	case *explorer.RemoteFile:
		return false, nil, v, nil
	default:
		return false, nil, nil, fmt.Errorf("unexpected type %T", v)
	}
}
