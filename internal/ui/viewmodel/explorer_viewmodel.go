package viewmodel

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
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

type ExplorerViewModel struct {
	mu                        sync.Mutex
	connRepo                  connection.Repository
	dirSvc                    *explorer.DirectoryService
	fileSvc                   *explorer.FileService
	tree                      binding.UntypedTree
	connections               binding.UntypedList // DEPRECATED
	state                     AppState
	loading                   binding.Bool
	lastSaveDir               fyne.ListableURI
	lastUploadDir             fyne.ListableURI
	errChan                   chan error
	displayNoConnectionBanner binding.Bool

	filesById map[explorer.S3FileID]*explorer.S3File
	dirsById  map[explorer.S3DirectoryID]*explorer.S3Directory
}

func NewExplorerViewModel(dirSvc *explorer.DirectoryService, connRepo connection.Repository, fileSvc *explorer.FileService) *ExplorerViewModel {
	t := binding.NewUntypedTree()
	c := binding.NewUntypedList()
	errChan := make(chan error)

	vm := &ExplorerViewModel{
		tree:                      t,
		dirSvc:                    dirSvc,
		connections:               c,
		state:                     NewAppState(),
		loading:                   binding.NewBool(),
		filesById:                 make(map[explorer.S3FileID]*explorer.S3File),
		dirsById:                  make(map[explorer.S3DirectoryID]*explorer.S3Directory),
		fileSvc:                   fileSvc,
		errChan:                   errChan,
		connRepo:                  connRepo,
		displayNoConnectionBanner: binding.NewBool(),
	}

	// Start error handler
	go func() {
		for err := range errChan {
			fmt.Printf("Error in ExplorerViewModel: %v\n", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := connRepo.GetSelectedConnection(ctx)
	if err != nil && err != connection.ErrConnectionNotFound {
		vm.errChan <- fmt.Errorf("error getting selected connection: %w", err)
	}
	if err == connection.ErrConnectionNotFound {
		vm.displayNoConnectionBanner.Set(true)
	} else {
		vm.displayNoConnectionBanner.Set(false)
		if err := vm.initializeTreeData(ctx); err != nil {
			vm.errChan <- fmt.Errorf("error resetting tree: %w", err)
		}
	}

	return vm
}

func (vm *ExplorerViewModel) OnDisplayNoConnectionBannerChange(fn func(shouldDisplay bool)) {
	vm.displayNoConnectionBanner.AddListener(binding.NewDataListener(func() {
		shouldDisplay, _ := vm.displayNoConnectionBanner.Get()
		fn(shouldDisplay)
	}))
}

func (vm *ExplorerViewModel) ErrorChan() chan error {
	return vm.errChan
}

func (vm *ExplorerViewModel) Loading() binding.Bool {
	return vm.loading
}

func (vm *ExplorerViewModel) StartLoading() {
	vm.loading.Set(true)
}

func (vm *ExplorerViewModel) StopLoading() {
	vm.loading.Set(false)
}

func (vm *ExplorerViewModel) Tree() binding.UntypedTree {
	return vm.tree
}

func (vm *ExplorerViewModel) AppendDirToTree(dirID explorer.S3DirectoryID) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	isDirAlreadyInTree := false
	_, err := vm.tree.GetValue(dirID.String())
	if err == nil {
		isDirAlreadyInTree = true
	}

	dir, err := vm.dirSvc.GetDirectoryByID(ctx, dirID)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting directory by ID (%s): %w", dirID, err)
		return err
	}
	vm.mu.Lock()
	vm.dirsById[dirID] = dir
	vm.mu.Unlock()

	fmt.Printf("Appending directory to tree: %s\n", dirID)
	dirNode := NewTreeNode(dirID.String(), dirID.ToName(), true)
	if isDirAlreadyInTree {
		if err := vm.tree.SetValue(dirID.String(), dirNode); err != nil {
			vm.errChan <- fmt.Errorf("error setting tree value: %w", err)
			return err
		}
	} else {
		if err := vm.tree.Append(dirID.String(), dirNode.ID, dirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending directory to tree: %w", err)
			return err
		}
	}
	dirNode.Loaded = true

	for _, file := range dir.Files {
		fileNode := NewTreeNode(file.ID.String(), file.Name, false)
		if err := vm.tree.Append(dirID.String(), fileNode.ID, fileNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
			continue
		}
		fileNode.Loaded = true
		vm.mu.Lock()
		vm.filesById[file.ID] = file
		vm.mu.Unlock()
	}
	for _, subDirID := range dir.SubDirectoriesIDs {
		subDirNode := NewTreeNode(subDirID.String(), subDirID.ToName(), true)
		if err := vm.tree.Append(dirID.String(), subDirNode.ID, subDirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
			continue
		}
		subDirNode.Loaded = false
	}

	return nil
}

func (vm *ExplorerViewModel) GetDirByID(dirID explorer.S3DirectoryID) (*explorer.S3Directory, error) {
	vm.mu.Lock()
	dir, ok := vm.dirsById[dirID]
	vm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("directory not found in cache")
	}
	return dir, nil
}

func (vm *ExplorerViewModel) GetFileByID(fileID explorer.S3FileID) (*explorer.S3File, error) {
	vm.mu.Lock()
	file, ok := vm.filesById[fileID]
	vm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("file not found in cache")
	}
	return file, nil
}

func (vm *ExplorerViewModel) PreviewFile(f *explorer.S3File) (string, error) {
	if f.SizeBytes > vm.GetMaxFileSizePreview() {
		return "", fmt.Errorf("file is too big to PreviewFile")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	content, err := vm.fileSvc.GetContent(ctx, f)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting file content: %w", err)
		return "", err
	}

	return string(content), nil
}

func (vm *ExplorerViewModel) GetMaxFileSizePreview() int64 {
	return 1024 * 1024
}

func (vm *ExplorerViewModel) SelectedConnection() *connection.Connection {
	return vm.state.SelectedConnection
}

func (vm *ExplorerViewModel) ResetTree() error {
	vm.resetTreeContent()
	return vm.initializeTreeData(context.Background())
}

func (vm *ExplorerViewModel) DownloadFile(f *explorer.S3File, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := vm.fileSvc.DownloadFile(ctx, f, dest); err != nil {
		vm.errChan <- fmt.Errorf("error downloading file: %w", err)
		return err
	}
	return nil
}

func (vm *ExplorerViewModel) UploadFile(localPath string, remoteDir *explorer.S3Directory) error {
	localFile := explorer.NewLocalFile(localPath)
	remoteFile, err := explorer.NewS3File(localFile.FileName(), remoteDir)
	if err != nil {
		vm.errChan <- fmt.Errorf("error creating S3 file: %w", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := vm.fileSvc.UploadFile(ctx, localFile, remoteFile); err != nil {
		vm.errChan <- fmt.Errorf("error uploading file: %w", err)
		return err
	}

	return nil
}

func (vm *ExplorerViewModel) GetLastSaveDir() fyne.ListableURI {
	return vm.lastSaveDir
}

func (vm *ExplorerViewModel) SetLastSaveDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SaveLastDir: %w", err)
	}
	vm.lastSaveDir = uriLister
	return nil
}

func (vm *ExplorerViewModel) GetLastUploadDir() fyne.ListableURI {
	return vm.lastUploadDir
}

func (vm *ExplorerViewModel) SetLastUploadDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SetLastUploadDir: %w", err)
	}
	vm.lastUploadDir = uriLister
	return nil
}

func (vm *ExplorerViewModel) resetTreeContent() {
	vm.tree = binding.NewUntypedTree()
}

func (vm *ExplorerViewModel) initializeTreeData(ctx context.Context) error {
	rootDir, err := vm.dirSvc.GetRootDirectory(ctx)
	if err != nil {
		return fmt.Errorf("error getting root directory: %w", err)
	}
	rootNode := NewTreeNode(rootDir.ID.String(), rootDir.ID.ToName(), true)
	if err := vm.tree.Append("", rootNode.ID, rootNode); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	if err := vm.AppendDirToTree(rootDir.ID); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	return nil
}
