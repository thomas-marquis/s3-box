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
	errChan := make(chan error)

	vm := &ExplorerViewModel{
		tree:                      t,
		dirSvc:                    dirSvc,
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout*2)
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

func (vm *ExplorerViewModel) RefreshDir(dirID explorer.S3DirectoryID) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dir, err := vm.fetchAndUpdateDirectory(ctx, dirID)
	if err != nil {
		return err
	}

	if err := vm.removeDirectoryContent(dirID); err != nil {
		return err
	}

	return vm.appendDirectoryContent(dirID, dir)
}

func (vm *ExplorerViewModel) fetchAndUpdateDirectory(ctx context.Context, dirID explorer.S3DirectoryID) (*explorer.S3Directory, error) {
	dir, err := vm.dirSvc.GetDirectoryByID(ctx, dirID)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting directory by ID (%s): %w", dirID, err)
		return nil, err
	}

	vm.mu.Lock()
	vm.dirsById[dirID] = dir
	vm.mu.Unlock()

	return dir, nil
}

func (vm *ExplorerViewModel) removeDirectoryContent(dirID explorer.S3DirectoryID) error {
	oldDir, err := vm.GetDirByID(dirID)
	if err != nil {
		return nil
	}

	for _, file := range oldDir.Files {
		if err := vm.tree.Remove(file.ID.String()); err != nil {
			vm.errChan <- fmt.Errorf("error removing old file from tree: %w", err)
		}
	}

	for _, subDirID := range oldDir.SubDirectoriesIDs {
		if err := vm.tree.Remove(subDirID.String()); err != nil {
			vm.errChan <- fmt.Errorf("error removing old subdirectory from tree: %w", err)
		}
	}

	return nil
}

func (vm *ExplorerViewModel) appendDirectoryContent(dirID explorer.S3DirectoryID, dir *explorer.S3Directory) error {
	for _, file := range dir.Files {
		if err := vm.appendFileNode(dirID, file); err != nil {
			continue
		}
	}

	for _, subDirID := range dir.SubDirectoriesIDs {
		if err := vm.appendDirectoryNode(dirID, subDirID); err != nil {
			continue
		}
	}

	return nil
}

func (vm *ExplorerViewModel) appendFileNode(parentDirID explorer.S3DirectoryID, file *explorer.S3File) error {
	fileNode := NewTreeNode(file.ID.String(), file.Name, false)
	fileNode.Loaded = true

	if err := vm.tree.Append(parentDirID.String(), fileNode.ID, fileNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
		return err
	}

	vm.mu.Lock()
	vm.filesById[file.ID] = file
	vm.mu.Unlock()

	return nil
}

func (vm *ExplorerViewModel) appendDirectoryNode(parentDirID explorer.S3DirectoryID, dirID explorer.S3DirectoryID) error {
	dirNode := NewTreeNode(dirID.String(), dirID.ToName(), true)
	dirNode.Loaded = false

	if err := vm.tree.Append(parentDirID.String(), dirNode.ID, dirNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
		return err
	}

	return nil
}

func (vm *ExplorerViewModel) AppendDirToTree(dirID explorer.S3DirectoryID) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	di, err := vm.tree.GetValue(dirID.String())
	var existingNode *TreeNode = nil
	if err == nil {
		var ok bool
		existingNode, ok = di.(*TreeNode)
		if !ok {
			panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirID.String()))
		}
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
	if existingNode != nil {
		existingNode.Loaded = true
	} else {
		dirNode := NewTreeNode(dirID.String(), dirID.ToName(), true)
		dirNode.Loaded = true
		if err := vm.tree.Append(dirID.String(), dirNode.ID, dirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending directory to tree: %w", err)
			return err
		}
	}

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

func (vm *ExplorerViewModel) RemoveDirToTree(dirID explorer.S3DirectoryID) error {
	item, err := vm.tree.GetValue(dirID.String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the direcotry you want to remove: %s", dirID.String())
	}
	node, ok := item.(*TreeNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirID.String()))
	}

	if !node.Loaded {
		if err := vm.tree.Remove(dirID.String()); err != nil {
			vm.errChan <- fmt.Errorf("error removing directory from tree: %w", err)
			return err
		}
	} else {
		vm.mu.Lock()
		dir, ok := vm.dirsById[dirID]
		vm.mu.Unlock()
		if !ok {
			panic(fmt.Sprintf("impossible to find the directory in the cache: %s", dirID.String()))
		}
		for _, f := range dir.Files {
			if err := vm.tree.Remove(f.ID.String()); err != nil {
				vm.errChan <- fmt.Errorf("error removing file from tree: %w", err)
				return err
			}
		}
		for _, subDirID := range dir.SubDirectoriesIDs {
			if err := vm.RemoveDirToTree(subDirID); err != nil {
				vm.errChan <- fmt.Errorf("error removing subdirectory from tree: %w", err)
				return err
			}
		}
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

	return vm.RefreshDir(remoteDir.ID)
}

func (vm *ExplorerViewModel) DeleteFile(file *explorer.S3File) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Get the parent directory
	dir, err := vm.dirSvc.GetDirectoryByID(ctx, file.DirectoryID)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting parent directory: %w", err)
		return err
	}

	// Delete the file using the DirectoryService
	if err := vm.dirSvc.DeleteFile(ctx, dir, file.ID); err != nil {
		vm.errChan <- fmt.Errorf("error deleting file: %w", err)
		return err
	}

	// Remove file from tree
	if err := vm.tree.Remove(file.ID.String()); err != nil {
		vm.errChan <- fmt.Errorf("error removing file from tree: %w", err)
		return err
	}

	// Remove file from cache
	vm.mu.Lock()
	delete(vm.filesById, file.ID)
	vm.mu.Unlock()

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
	currentConn, err := vm.connRepo.GetSelectedConnection(ctx)
	if err != nil {
		return fmt.Errorf("error getting selected connection: %w", err)
	}
	displayLabel := "Bucket: " + currentConn.BucketName
	rootNode := NewTreeNode(rootDir.ID.String(), displayLabel, true)
	if err := vm.tree.Append("", rootNode.ID, rootNode); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	if err := vm.AppendDirToTree(rootDir.ID); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	return nil
}
