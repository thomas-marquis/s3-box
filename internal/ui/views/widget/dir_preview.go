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
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type DirectoryPreview struct {
	widget.BaseWidget

	OnValidate func(strategy directory.MaterializeStrategy)

	appCtx           appcontext.AppContext
	data             binding.Tree[previewNodeItem]
	preview          *directory.Preview
	infoData         binding.String
	selectedStrategy binding.Item[directory.MaterializeStrategy]
}

func NewDirectoryPreview(appCtx appcontext.AppContext, preview *directory.Preview) *DirectoryPreview {
	data := binding.NewTree[previewNodeItem](func(p1, p2 previewNodeItem) bool {
		return p1 == p2
	})

	initPreviewData(data, preview, "")

	w := &DirectoryPreview{
		appCtx:     appCtx,
		data:       data,
		preview:    preview,
		infoData:   binding.NewString(),
		OnValidate: func(directory.MaterializeStrategy) {},
		selectedStrategy: binding.NewItem[directory.MaterializeStrategy](func(s1, s2 directory.MaterializeStrategy) bool {
			return s1 == s2
		}),
	}

	w.ExtendBaseWidget(w)

	return w
}

func (w *DirectoryPreview) makeContent(mainContent fyne.CanvasObject, validateLabel string) fyne.CanvasObject {
	var validateBtn fyne.CanvasObject
	onValidate := func() {
		val, _ := w.selectedStrategy.Get()
		w.OnValidate(val)
	}
	if validateLabel != "" {
		validateBtn = widget.NewButton(validateLabel, onValidate)
	} else {
		validateBtn = NewButtonWithData(
			uiutils.NewBindingItemFormatter(w.selectedStrategy, func(strategy directory.MaterializeStrategy) string {
				return "Upload: " + strategy.String()
			}),
			onValidate)
	}

	return container.NewBorder(
		container.NewBorder(nil, nil,
			widget.NewLabel(fmt.Sprintf("You are about to upload %d directories and %d files",
				w.preview.Count().Directories, w.preview.Count().Files)),
			widget.NewLabelWithData(w.infoData)),
		container.NewBorder(nil, nil, nil,
			validateBtn,
		),
		nil, nil,
		mainContent,
	)
}

func (w *DirectoryPreview) CreateRenderer() fyne.WidgetRenderer {
	w.ExtendBaseWidget(w)

	strategies := w.preview.AvailableStrategies()
	if len(strategies) == 0 {
		panic("no strategies available")
	}

	w.selectedStrategy.Set(strategies[0]) //nolint:errcheck
	if len(strategies) == 1 {
		return widget.NewSimpleRenderer(w.makeContent(
			w.makeTree(strategies[0]),
			"Upload"))
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
	tabs.OnSelected = func(item *container.TabItem) {
		w.infoData.Set("")                                       //nolint:errcheck
		w.selectedStrategy.Set(strategies[tabs.SelectedIndex()]) //nolint:errcheck
	}

	return widget.NewSimpleRenderer(w.makeContent(tabs, ""))
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
				dir := ni.Preview.Directory()
				if dir.Path() == directory.RootPath {
					icon.SetResource(theme.StorageIcon())
					label.SetText("Bucket root")
				} else {
					icon.SetResource(theme.FolderIcon())
					if dir.Is(ni.Preview.MountPoint()) {
						label.SetText(dir.Path().String())
					} else {
						val := dir.Name()
						status := ni.Preview.DirStatus()
						if status != "" {
							val += " (" + status + ")"
						}
						label.SetText(val)
					}
				}
				infoBtn.Hide()
			} else {
				icon.SetResource(theme.FileIcon())

				fileName := ni.File.Name().String()
				fileLabelBuilder := strings.Builder{}
				fileLabelBuilder.WriteString(fileName)

				status, desc, err := ni.Preview.FileStatus(strategy, fileName)
				if err != nil {
					fmt.Fprintf(&fileLabelBuilder, " (%s)", err) //nolint:errcheck
				}
				fmt.Fprintf(&fileLabelBuilder, " (%s)", status) //nolint:errcheck

				if desc != "" {
					infoBtn.Show()
					infoBtn.OnTapped = func() {
						w.infoData.Set(desc) //nolint:errcheck
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
	data.Append(parentPath, currPath, previewNodeItem{ // nolint:errcheck
		Preview: preview,
		File:    nil,
	})
	for _, f := range preview.Files() {
		data.Append(currPath, currPath+f.Name().String(), previewNodeItem{ // nolint:errcheck
			Preview: preview,
			File:    f,
		})
	}
	for _, c := range preview.Children() {
		initPreviewData(data, c, currPath)
	}
}
