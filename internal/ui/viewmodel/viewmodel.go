package viewmodel

import (
	"context"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
)

const (
	timeout = 15 * time.Second
)

type ViewModel struct {
	explorerRepo  explorer.Repository
	connRepo      connection.Repository
	dirSvc        *explorer.DirectoryService
	tree          binding.UntypedTree
	connections   binding.UntypedList
	state         AppState
	loading       binding.Bool
	lastSaveDir   fyne.ListableURI
	lastUploadDir fyne.ListableURI
}

func NewViewModel(explorerRepo explorer.Repository, dirScv *explorer.DirectoryService, connRepo connection.Repository) *ViewModel {
	t := binding.NewUntypedTree()
	t.Append("", explorer.RootDir.Path(), explorer.RootDir)

	c := binding.NewUntypedList()

	vm := &ViewModel{
		explorerRepo: explorerRepo,
		tree:         t,
		dirSvc:       dirScv,
		connections:  c,
		connRepo:     connRepo,
		state:        NewAppState(),
		loading:      binding.NewBool(),
	}

	if err := vm.RefreshConnections(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	currentConn, err := connRepo.GetSelectedConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		panic(err)
	}
	if currentConn != nil {
		vm.state.SelectedConnection = currentConn
	}

	return vm
}

func (vm *ViewModel) Loading() binding.Bool {
	return vm.loading
}

func (vm *ViewModel) StartLoading() {
	vm.loading.Set(true)
}

func (vm *ViewModel) StopLoading() {
	vm.loading.Set(false)
}

func (vm *ViewModel) Tree() binding.UntypedTree {
	return vm.tree
}

func (vm *ViewModel) ExpandDir(d *explorer.S3Directory) error {
	if d.IsLoaded {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.dirSvc.Load(ctx, d); err != nil {
		if err == explorer.ErrConnectionNoSet {
			return ErrNoConnectionSelected
		}
		return err
	}

	for _, sd := range d.SubDirectories {
		vm.tree.Append(d.Path(), sd.Path(), sd)
	}
	for _, f := range d.Files {
		vm.tree.Append(d.Path(), f.Path(), f)
	}

	return nil
}

func (vm *ViewModel) RefreshDir(d *explorer.S3Directory) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.dirSvc.Load(ctx, d); err != nil {
		if err == explorer.ErrConnectionNoSet {
			return ErrNoConnectionSelected
		}
		return err
	}

	for _, sd := range d.SubDirectories {
		vm.tree.Remove(sd.Path())
		vm.tree.Append(d.Path(), sd.Path(), sd)
	}
	for _, f := range d.Files {
		vm.tree.Remove(f.Path())
		vm.tree.Append(d.Path(), f.Path(), f)
	}

	return nil
}

func (vm *ViewModel) Connections() binding.UntypedList {
	return vm.connections
}

func (vm *ViewModel) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conns, err := vm.connRepo.ListConnections(ctx)
	if err != nil {
		return err
	}

	prevConns, err := vm.connections.Get()
	if err != nil {
		return err
	}
	for _, c := range prevConns {
		vm.connections.Remove(c)
	}

	for _, c := range conns {
		vm.connections.Append(c)
	}

	return nil
}

func (vm *ViewModel) SaveConnection(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.connRepo.SaveConnection(ctx, c); err != nil {
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ViewModel) DeleteConenction(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.connRepo.DeleteConnection(ctx, c.ID); err != nil {
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ViewModel) PreviewFile(f *explorer.S3File) (string, error) {
	if f.SizeBytes() > vm.GetMaxFileSizePreview() {
		return "", fmt.Errorf("file is too big to PreviewFile")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	content, err := vm.explorerRepo.GetFileContent(ctx, f)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (vm *ViewModel) GetMaxFileSizePreview() int64 {
	return 1024 * 1024
}

func (vm *ViewModel) SelectedConnection() *connection.Connection {
	return vm.state.SelectedConnection
}

func (vm *ViewModel) SelectConnection(c *connection.Connection) error {
	vm.loading.Set(true)
	prevConn := vm.state.SelectedConnection

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	if err := vm.connRepo.SetSelectedConnection(ctx, c.ID); err != nil {
		cancel()
		return err
	}
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.explorerRepo.SetConnection(ctx, c); err != nil {
		if err := vm.connRepo.SetSelectedConnection(ctx, prevConn.ID); err != nil {
			return err
		}
		return err
	}

	if c != prevConn {
		vm.resetTreeContent()
		explorer.RootDir.IsLoaded = false // TODO: crado, crecréer un rootdir plutôt
		explorer.RootDir.SubDirectories = make([]*explorer.S3Directory, 0)
		explorer.RootDir.Files = make([]*explorer.S3File, 0)
		if err := vm.tree.Append("", explorer.RootDir.Path(), explorer.RootDir); err != nil {
			return err
		}
		if err := vm.ExpandDir(explorer.RootDir); err != nil {
			return err
		}
	}

	vm.state.SelectedConnection = c
	vm.loading.Set(false)
	return nil
}

func (vm *ViewModel) DownloadFile(f *explorer.S3File, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return vm.explorerRepo.DownloadFile(ctx, f, dest)
}

func (vm *ViewModel) UploadFile(localPath string, remoteDir *explorer.S3Directory) error {
	localFile := explorer.NewLocalFile(localPath)
	remoteFile := explorer.NewS3File(remoteDir.Path() + "/" + localFile.FileName())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := vm.explorerRepo.UploadFile(ctx, localFile, remoteFile); err != nil {
		return fmt.Errorf("UploadFile: %w", err)
	}

	return nil
}

func (vm *ViewModel) GetLastSaveDir() fyne.ListableURI {
	return vm.lastSaveDir
}

func (vm *ViewModel) SetLastSaveDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SaveLastDir: %w", err)
	}
	vm.lastSaveDir = uriLister
	return nil
}

func (vm *ViewModel) GetLastUploadDir() fyne.ListableURI {
	return vm.lastUploadDir
}

func (vm *ViewModel) SetLastUploadDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SetLastUploadDir: %w", err)
	}
	vm.lastUploadDir = uriLister
	return nil
}

func (vm *ViewModel) resetTreeContent() {
	vm.tree = binding.NewUntypedTree()
}
