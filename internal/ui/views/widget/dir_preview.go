package widget

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	appcontext "github.com/thomas-marquis/s3-box/internal/ui/app/context"
)

type DirectoryPreview struct {
	widget.BaseWidget

	appCtx   appcontext.AppContext
	data     binding.Tree[previewNodeItem]
	preview  *directory.Preview
	infoData binding.String
}

func NewDirectoryPreview(appCtx appcontext.AppContext, preview *directory.Preview) *DirectoryPreview {
	data := binding.NewTree[previewNodeItem](func(p1, p2 previewNodeItem) bool {
		return p1 == p2
	})

	initPreviewData(data, preview, "")

	w := &DirectoryPreview{
		appCtx:   appCtx,
		data:     data,
		preview:  preview,
		infoData: binding.NewString(),
	}

	w.ExtendBaseWidget(w)

	return w
}

func (w *DirectoryPreview) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	strategies := w.preview.AvailableStrategies()
	if len(strategies) == 1 {
		return widget.NewSimpleRenderer(container.NewBorder(
			widget.NewLabelWithData(w.infoData), nil, nil, nil,
			w.makeTree(strategies[0]),
		))
	}

	var tis []*container.TabItem
	for _, s := range strategies {
		tis = append(tis, container.NewTabItem(
			s.String(),
			w.makeTree(s),
		))
	}
	tabs := container.NewAppTabs(
		tis...,
	)
	tabs.OnSelected = func(*container.TabItem) {
		w.infoData.Set("")
	}

	return widget.NewSimpleRenderer(container.NewBorder(
		widget.NewLabelWithData(w.infoData), nil, nil, nil,
		tabs,
	))
}

func (w *DirectoryPreview) makeTree(strategy directory.MaterializeStrategy) *widget.Tree {
	tree := widget.NewTreeWithData(w.data,
		func(b bool) fyne.CanvasObject {
			icon := widget.NewIcon(theme.FolderIcon())
			label := widget.NewLabel("")
			infoBtn := widget.NewButtonWithIcon("", theme.QuestionIcon(), func() {})
			infoBtn.Hide()
			return container.NewHBox(icon, label, infoBtn)
		},
		func(item binding.DataItem, b bool, object fyne.CanvasObject) {
			ni, err := item.(binding.Item[previewNodeItem]).Get()
			if err != nil {
				panic(fmt.Errorf("unexpected type %T: %w", ni, err))
			}

			icon := object.(*fyne.Container).Objects[0].(*widget.Icon)
			label := object.(*fyne.Container).Objects[1].(*widget.Label)
			infoBtn := object.(*fyne.Container).Objects[2].(*widget.Button)

			if ni.IsDir() {
				icon.SetResource(theme.FolderIcon())
				dirName := ni.Preview.Directory().Name()
				label.SetText(dirName)
				infoBtn.Hide()
			} else {
				icon.SetResource(theme.FileIcon())

				fileName := ni.File.Name().String()
				fileLabelBuilder := strings.Builder{}
				fileLabelBuilder.WriteString(fileName)

				status, desc, err := ni.Preview.FileStatus(strategy, fileName)
				if err != nil {
					fileLabelBuilder.WriteString(fmt.Sprintf(" (%s)", err))
				}
				fileLabelBuilder.WriteString(fmt.Sprintf(" (%s)", status))

				if desc != "" {
					infoBtn.Show()
					infoBtn.OnTapped = func() {
						w.infoData.Set(desc)
					}
				} else {
					infoBtn.Hide()
				}

				label.SetText(fileLabelBuilder.String())
			}

			icon.Show()
			label.Show()
		},
	)
	tree.OpenAllBranches()
	return tree
}

type previewNodeItem struct {
	Preview *directory.Preview
	File    *directory.File
}

func (i *previewNodeItem) IsDir() bool {
	return i.File == nil
}

func initPreviewData(data binding.Tree[previewNodeItem], preview *directory.Preview, parentPath string) {
	currPath := parentPath + preview.Directory().Name() + "/"
	data.Append(parentPath, currPath, previewNodeItem{
		Preview: preview,
		File:    nil,
	})
	for _, f := range preview.Files() {
		data.Append(currPath, currPath+f.Name().String(), previewNodeItem{
			Preview: preview,
			File:    f,
		})
	}
	for _, c := range preview.Children() {
		initPreviewData(data, c, currPath)
	}
}
