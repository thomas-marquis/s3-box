package viewmodel

import (
	"context"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/ui/viewerror"
)

type ExplorerViewModel struct {
	explorerRepo       explorer.Repository
	connRepo           connection.Repository
	dirSvc             *explorer.DirectoryService
	tree               binding.UntypedTree
	connections        binding.UntypedList
	loading            binding.Bool
	lastSaveDir        fyne.ListableURI
	lastUploadDir      fyne.ListableURI
	selectedConnection *connection.Connection
}

func NewExplorerViewModel(explorerRepo explorer.Repository, dirScv *explorer.DirectoryService, connRepo connection.Repository) *ExplorerViewModel {
	t := binding.NewUntypedTree()
	t.Append("", explorer.RootDir.Path(), explorer.RootDir)

	c := binding.NewUntypedList()

	vm := &ExplorerViewModel{
		explorerRepo: explorerRepo,
		tree:         t,
		dirSvc:       dirScv,
		connections:  c,
		connRepo:     connRepo,
		loading:      binding.NewBool(),
	}

	// if err := vm.RefreshConnections(); err != nil {
	// 	panic(err)
	// }

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	currentConn, err := connRepo.GetSelectedConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		panic(err)
	}
	if currentConn != nil {
		vm.selectedConnection = currentConn
	}

	return vm
}

func (v *ExplorerViewModel) Loading() binding.Bool {
	return v.loading
}

func (v *ExplorerViewModel) Tree() binding.UntypedTree {
	return v.tree
}

func (v *ExplorerViewModel) StartLoading() {
	v.loading.Set(true)
}

func (v *ExplorerViewModel) StopLoading() {
	v.loading.Set(false)
}
func (v *ExplorerViewModel) ExpandDir(d *explorer.Directory) error {
	if d.IsLoaded {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := v.dirSvc.Load(ctx, d); err != nil {
		if err == explorer.ErrConnectionNoSet {
			return viewerror.ErrNoConnectionSelected
		}
		return err
	}

	for _, sd := range d.SubDirectories {
		v.tree.Append(d.Path(), sd.Path(), sd)
	}
	for _, f := range d.Files {
		v.tree.Append(d.Path(), f.Path(), f)
	}

	return nil
}

func (v *ExplorerViewModel) RefreshDir(d *explorer.Directory) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := v.dirSvc.Load(ctx, d); err != nil {
		if err == explorer.ErrConnectionNoSet {
			return viewerror.ErrNoConnectionSelected
		}
		return err
	}

	for _, sd := range d.SubDirectories {
		v.tree.Remove(sd.Path())
		v.tree.Append(d.Path(), sd.Path(), sd)
	}
	for _, f := range d.Files {
		v.tree.Remove(f.Path())
		v.tree.Append(d.Path(), f.Path(), f)
	}

	return nil
}

func (v *ExplorerViewModel) PreviewFile(f *explorer.RemoteFile) (string, error) {
	if f.SizeBytes() > v.GetMaxFileSizePreview() {
		return "", fmt.Errorf("file is too big to PreviewFile")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	content, err := v.explorerRepo.GetFileContent(ctx, f)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (*ExplorerViewModel) GetMaxFileSizePreview() int64 {
	return 1024 * 1024
}

func (v *ExplorerViewModel) DownloadFile(f *explorer.RemoteFile, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return v.explorerRepo.DownloadFile(ctx, f, dest)
}

func (v *ExplorerViewModel) UploadFile(localPath string, remoteDir *explorer.Directory) error {
	localFile := explorer.NewLocalFile(localPath)
	remoteFile := explorer.NewRemoteFile(remoteDir.Path()+"/"+localFile.FileName(), remoteDir)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := v.explorerRepo.UploadFile(ctx, localFile, remoteFile); err != nil {
		return fmt.Errorf("UploadFile: %w", err)
	}

	return nil
}

func (v *ExplorerViewModel) GetLastSaveDir() fyne.ListableURI {
	return v.lastSaveDir
}

func (v *ExplorerViewModel) SetLastSaveDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SaveLastDir: %w", err)
	}
	v.lastSaveDir = uriLister
	return nil
}

func (v *ExplorerViewModel) GetLastUploadDir() fyne.ListableURI {
	return v.lastUploadDir
}

func (v *ExplorerViewModel) SetLastUploadDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SetLastUploadDir: %w", err)
	}
	v.lastUploadDir = uriLister
	return nil
}

func (vm *ExplorerViewModel) resetTreeContent() {
	vm.tree = binding.NewUntypedTree()
}

func (v *ExplorerViewModel) DeleteFile(f *explorer.RemoteFile) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return v.explorerRepo.DeleteFile(ctx, f)
}

func (v *ExplorerViewModel) ResetTree() error {
	v.resetTreeContent()

	explorer.RootDir.IsLoaded = false // TODO: crado, recréer un rootdir plutôt
	explorer.RootDir.SubDirectories = make([]*explorer.Directory, 0)
	explorer.RootDir.Files = make([]*explorer.RemoteFile, 0)

	if err := v.tree.Append("", explorer.RootDir.Path(), explorer.RootDir); err != nil {
		return err
	}
	if err := v.ExpandDir(explorer.RootDir); err != nil {
		return err
	}

	return nil
}
