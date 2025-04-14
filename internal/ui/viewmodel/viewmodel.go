package viewmodel

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
)

const (
	timeout = 15 * time.Second
)

type ViewModel struct {
	explorerRepo  explorer.S3DirectoryRepository
	connRepo      connection.Repository
	dirSvc        *explorer.DirectoryService
	fileSvc       *explorer.FileService
	tree          binding.UntypedTree
	connections   binding.UntypedList
	state         AppState
	loading       binding.Bool
	lastSaveDir   fyne.ListableURI
	lastUploadDir fyne.ListableURI
	errChan       chan error

	filesById map[explorer.S3FileID]*explorer.S3File
	dirsById  map[explorer.S3DirectoryID]*explorer.S3Directory
}

func NewViewModel(explorerRepo explorer.S3DirectoryRepository, dirSvc *explorer.DirectoryService, connRepo connection.Repository, fileSvc *explorer.FileService) *ViewModel {
	t := binding.NewUntypedTree()
	c := binding.NewUntypedList()
	errChan := make(chan error)

	vm := &ViewModel{
		tree:         t,
		dirSvc:       dirSvc,
		connections:  c,
		state:        NewAppState(),
		loading:      binding.NewBool(),
		filesById:    make(map[explorer.S3FileID]*explorer.S3File),
		dirsById:     make(map[explorer.S3DirectoryID]*explorer.S3Directory),
		fileSvc:      fileSvc,
		errChan:      errChan,
	}

	// Start error handler
	go func() {
		for err := range errChan {
			fmt.Printf("Error in ViewModel: %v\n", err)
		}
	}()

	rootNode := NewTreeNode(explorer.RootDirID.String(), "/", true)
	t.Append("", rootNode.ID, rootNode)
	// if err := vm.AppendDirToTree(context.Background(), explorer.RootDirID); err != nil {

	// }

	if err := vm.RefreshConnections(); err != nil {
		vm.errChan <- fmt.Errorf("Error refreshing connections: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	currentConn, err := vm.getCurrentConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		vm.errChan <- fmt.Errorf("Error getting selected connection: %w", err)
	}
	if currentConn != nil {
		vm.state.SelectedConnection = currentConn
		vm.setActiveConnection(currentConn)
	}

	return vm
}

func (vm *ViewModel) getCurrentConnection(ctx context.Context) (*connection.Connection, error) {
	repo, err := vm.connRepoFactory.Get(ctx, vm.state.SelectedConnection)
	if err != nil {
		return nil, err
	}
	return repo.GetSelectedConnection(ctx)
}

func (vm *ViewModel) setActiveConnection(conn *connection.Connection) {
	vm.dirSvc.SetActiveConnection(conn)
	vm.fileSvc.SetActiveConnection(conn)
}

func (vm *ViewModel) ErrorChan() chan error {
	return vm.errChan
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

func (vm *ViewModel) AppendDirToTree(ctx context.Context, dirID explorer.S3DirectoryID) error {
	isDirAlreadyInTree := false
	_, err := vm.tree.GetValue(dirID.String())
	if err == nil {
		isDirAlreadyInTree = true
	}

	dir, err := vm.dirSvc.GetDirectoryByID(ctx, dirID)
	if err != nil {
		vm.errChan <- fmt.Errorf("Error getting directory by ID: %w", err)
		return err
	}
	vm.dirsById[dirID] = dir

	dirNode := NewTreeNode(dirID.String(), dirID.ToName(), true)
	if isDirAlreadyInTree {
		if err := vm.tree.SetValue(dirID.String(), dirNode); err != nil {
			vm.errChan <- fmt.Errorf("Error setting tree value: %w", err)
			return err
		}
	} else {
		if err := vm.tree.Append(dirID.String(), dirNode.ID, dirNode); err != nil {
			vm.errChan <- fmt.Errorf("Error appending directory to tree: %w", err)
			return err
		}
	}
	dirNode.Loaded = true

	for _, file := range dir.Files {
		fileNode := NewTreeNode(file.ID.String(), file.Name, false)
		if err := vm.tree.Append(dirID.String(), fileNode.ID, fileNode); err != nil {
			vm.errChan <- fmt.Errorf("Error appending file to tree: %w", err)
			continue
		}
		fileNode.Loaded = true
		vm.filesById[file.ID] = file
	}
	for _, subDirID := range dir.SubDirectoriesIDs {
		subDirNode := NewTreeNode(subDirID.String(), subDirID.ToName(), true)
		if err := vm.tree.Append(dirID.String(), subDirNode.ID, subDirNode); err != nil {
			vm.errChan <- fmt.Errorf("Error appending subdirectory to tree: %w", err)
			continue
		}
		subDirNode.Loaded = true
	}

	return nil
}

// func (vm *ViewModel) ExpandDir(d *explorer.S3Directory) error {
// 	if d.IsLoaded {
// 		return nil
// 	}
// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()
// 	if err := vm.dirSvc.Load(ctx, d); err != nil {
// 		if err == explorer.ErrConnectionNoSet {
// 			return ErrNoConnectionSelected
// 		}
// 		return err
// 	}

// 	for _, sd := range d.SubDirectories {
// 		vm.tree.Append(d.Path(), sd.Path(), sd)
// 	}
// 	for _, f := range d.Files {
// 		vm.tree.Append(d.Path(), f.Path(), f)
// 	}

// 	return nil
// }

// func (vm *ViewModel) RefreshDir(d *explorer.S3Directory) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()
// 	if err := vm.dirSvc.Load(ctx, d); err != nil {
// 		if err == explorer.ErrConnectionNoSet {
// 			return ErrNoConnectionSelected
// 		}
// 		return err
// 	}

// 	for _, sd := range d.SubDirectories {
// 		vm.tree.Remove(sd.Path())
// 		vm.tree.Append(d.Path(), sd.Path(), sd)
// 	}
// 	for _, f := range d.Files {
// 		vm.tree.Remove(f.Path())
// 		vm.tree.Append(d.Path(), f.Path(), f)
// 	}

// 	return nil
// }

func (vm *ViewModel) Connections() binding.UntypedList {
	return vm.connections
}

func (vm *ViewModel) RefreshConnections() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conns, err := vm.connRepo.ListConnections(ctx)
	if err != nil {
		vm.errChan <- fmt.Errorf("Error listing connections: %w", err)
		return err
	}

	prevConns, err := vm.connections.Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("Error getting previous connections: %w", err)
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
		vm.errChan <- fmt.Errorf("Error saving connection: %w", err)
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ViewModel) DeleteConnection(c *connection.Connection) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.connRepo.DeleteConnection(ctx, c.ID); err != nil {
		vm.errChan <- fmt.Errorf("Error deleting connection: %w", err)
		return err
	}

	return vm.RefreshConnections()
}

func (vm *ViewModel) PreviewFile(f *explorer.S3File) (string, error) {
	if f.SizeBytes > vm.GetMaxFileSizePreview() {
		return "", fmt.Errorf("file is too big to PreviewFile")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	content, err := vm.fileSvc.GetContent(ctx, f)
	if err != nil {
		vm.errChan <- fmt.Errorf("Error getting file content: %w", err)
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

	// Set the active connection in the directory service
	vm.setActiveConnection(c)
	if err := vm.connRepo.SetSelectedConnection(ctx, c.ID); err != nil {
		return err
	}

	if c != prevConn {
		vm.resetTreeContent()
		// explorer.RootDir.IsLoaded = false // TODO: crado, crecréer un rootdir plutôt
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
	if err := vm.fileSvc.DownloadFile(ctx, f, dest); err != nil {
		vm.errChan <- fmt.Errorf("Error downloading file: %w", err)
		return err
	}
	return nil
}

func (vm *ViewModel) UploadFile(localPath string, remoteDir *explorer.S3Directory) error {
	localFile := explorer.NewLocalFile(localPath)
	remoteFile, err := explorer.NewS3File(localFile.FileName(), remoteDir)
	if err != nil {
		vm.errChan <- fmt.Errorf("Error creating S3 file: %w", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := vm.fileSvc.UploadFile(ctx, localFile, remoteFile); err != nil {
		vm.errChan <- fmt.Errorf("Error uploading file: %w", err)
		return err
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
