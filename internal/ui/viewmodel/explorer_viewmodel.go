package viewmodel

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"github.com/thomas-marquis/s3-box/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
)

type ExplorerViewModel interface {
	OnDisplayNoConnectionBannerChange(fn func(shouldDisplay bool))
	ErrorChan() chan error
	Loading() binding.Bool
	StartLoading()
	StopLoading()
	Tree() binding.UntypedTree
	RefreshDir(dirID explorer.S3DirectoryID) error

	// OpenDirectory opens a directory in the tree and loads its content.
	// If the directory is already open, it will refresh its content.
	OpenDirectory(dirID explorer.S3DirectoryID) error

	RemoveDirToTree(dirID explorer.S3DirectoryID) error
	GetDirByID(dirID explorer.S3DirectoryID) (*explorer.S3Directory, error)
	GetFileByID(fileID explorer.S3FileID) (*explorer.S3File, error)
	PreviewFile(f *explorer.S3File) (string, error)

	// GetMaxFileSizePreview returns the max file size preview in bytes
	GetMaxFileSizePreview() int64

	ResetTree() error
	DownloadFile(f *explorer.S3File, dest string) error
	UploadFile(localPath string, remoteDir *explorer.S3Directory) error
	DeleteFile(file *explorer.S3File) error
	GetLastSaveDir() fyne.ListableURI
	SetLastSaveDir(filePath string) error
	GetLastUploadDir() fyne.ListableURI
	SetLastUploadDir(filePath string) error

	// CreateEmptySubDirectory creates an empty subdirectory in the given parent directory
	CreateEmptyDirectory(parent *explorer.S3Directory, name string) (*explorer.S3Directory, error)

	// IsReadOnly checks if the current connection is read-only
	// IsReadOnly() bool
}

type explorerViewModelImpl struct {
	mu                        sync.Mutex
	connRepo                  connection.Repository
	dirSvc                    explorer.DirectoryService
	fileSvc                   explorer.FileService
	settingsVm                SettingsViewModel
	tree                      binding.UntypedTree
	loading                   binding.Bool
	lastSaveDir               fyne.ListableURI
	lastUploadDir             fyne.ListableURI
	errChan                   chan error
	displayNoConnectionBanner binding.Bool

	filesById map[explorer.S3FileID]*explorer.S3File
	dirsById  map[explorer.S3DirectoryID]*explorer.S3Directory
}

var _ ExplorerViewModel = &explorerViewModelImpl{}

func NewExplorerViewModel(
	dirSvc explorer.DirectoryService,
	connRepo connection.Repository,
	fileSvc explorer.FileService,
	settingsVm SettingsViewModel,
) *explorerViewModelImpl {
	t := binding.NewUntypedTree()
	errChan := make(chan error)

	vm := &explorerViewModelImpl{
		tree:                      t,
		dirSvc:                    dirSvc,
		settingsVm:                settingsVm,
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

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout()*2)
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

func (vm *explorerViewModelImpl) OnDisplayNoConnectionBannerChange(fn func(shouldDisplay bool)) {
	vm.displayNoConnectionBanner.AddListener(binding.NewDataListener(func() {
		shouldDisplay, _ := vm.displayNoConnectionBanner.Get()
		fn(shouldDisplay)
	}))
}

func (vm *explorerViewModelImpl) ErrorChan() chan error {
	return vm.errChan
}

func (vm *explorerViewModelImpl) Loading() binding.Bool {
	return vm.loading
}

func (vm *explorerViewModelImpl) StartLoading() {
	vm.loading.Set(true)
}

func (vm *explorerViewModelImpl) StopLoading() {
	vm.loading.Set(false)
}

func (vm *explorerViewModelImpl) Tree() binding.UntypedTree {
	return vm.tree
}

func (vm *explorerViewModelImpl) RefreshDir(dirID explorer.S3DirectoryID) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	dir, err := vm.fetchAndUpdateDirectory(ctx, dirID)
	if err != nil {
		return err
	}

	dirTreeNodeItem, err := vm.tree.GetValue(dirID.String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the direcotry you want to refresh: %s", dirID.String())
	}
	dirTreeNode, ok := dirTreeNodeItem.(*TreeNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirID.String()))
	}
	dirTreeNode.SetIsLoaded()

	if err := vm.removeDirectoryContent(dirID); err != nil {
		return err
	}

	return vm.appendDirectoryContent(dirID, dir)
}

func (vm *explorerViewModelImpl) OpenDirectory(dirID explorer.S3DirectoryID) error {
	di, err := vm.tree.GetValue(dirID.String())
	var existingNode *TreeNode = nil
	if err == nil {
		var ok bool
		existingNode, ok = di.(*TreeNode)
		if !ok {
			panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirID.String()))
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, vm.settingsVm.CurrentTimeout())
	defer cancel()

	dir, err := vm.dirSvc.GetDirectoryByID(ctx, dirID)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting directory by ID (%s): %w", dirID, err)
		return err
	}
	vm.mu.Lock()
	vm.dirsById[dirID] = dir
	vm.mu.Unlock()

	if existingNode != nil {
		existingNode.SetIsLoaded()
	} else {
		dirNode := NewTreeNode(dirID.String(), dirID.ToName(), TreeNodeTypeDirectory)
		dirNode.SetIsLoaded()
		if err := vm.tree.Append(dirID.String(), dirNode.ID, dirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending directory to tree: %w", err)
			return err
		}
	}

	for _, file := range dir.Files {
		fileNode := NewTreeNode(file.ID.String(), file.Name, TreeNodeTypeFile)
		if err := vm.tree.Append(dirID.String(), fileNode.ID, fileNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
			continue
		}
		fileNode.SetIsLoaded()
		vm.mu.Lock()
		vm.filesById[file.ID] = file
		vm.mu.Unlock()
	}
	for _, subDirID := range dir.SubDirectoriesIDs {
		subDirNode := NewTreeNode(subDirID.String(), subDirID.ToName(), TreeNodeTypeDirectory)
		if err := vm.tree.Append(dirID.String(), subDirNode.ID, subDirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
			continue
		}
		subDirNode.SetIsNotLoaded()
	}

	return nil
}

func (vm *explorerViewModelImpl) RemoveDirToTree(dirID explorer.S3DirectoryID) error {
	item, err := vm.tree.GetValue(dirID.String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the direcotry you want to remove: %s", dirID.String())
	}
	node, ok := item.(*TreeNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirID.String()))
	}

	if !node.IsLoaded() {
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

func (vm *explorerViewModelImpl) GetDirByID(dirID explorer.S3DirectoryID) (*explorer.S3Directory, error) {
	vm.mu.Lock()
	dir, ok := vm.dirsById[dirID]
	vm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("directory not found in cache")
	}
	return dir, nil
}

func (vm *explorerViewModelImpl) GetFileByID(fileID explorer.S3FileID) (*explorer.S3File, error) {
	vm.mu.Lock()
	file, ok := vm.filesById[fileID]
	vm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("file not found in cache")
	}
	return file, nil
}

func (vm *explorerViewModelImpl) PreviewFile(f *explorer.S3File) (string, error) {
	if f.SizeBytes > vm.GetMaxFileSizePreview() {
		return "", fmt.Errorf("file is too big to PreviewFile")
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	content, err := vm.fileSvc.GetContent(ctx, f)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting file content: %w", err)
		return "", err
	}

	return string(content), nil
}

func (vm *explorerViewModelImpl) GetMaxFileSizePreview() int64 {
	val, err := vm.settingsVm.MaxFilePreviewSizeMegaBytes().Get()
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting max file size preview: %w", err)
		return 0
	}
	return utils.MegaToBytes(int64(val))
}

func (vm *explorerViewModelImpl) ResetTree() error {
	vm.resetTreeContent()
	return vm.initializeTreeData(context.Background())
}

func (vm *explorerViewModelImpl) DownloadFile(f *explorer.S3File, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	if err := vm.fileSvc.DownloadFile(ctx, f, dest); err != nil {
		vm.errChan <- fmt.Errorf("error downloading file: %w", err)
		return err
	}
	return nil
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, remoteDir *explorer.S3Directory) error {
	localFile := explorer.NewLocalFile(localPath)
	remoteFile, err := explorer.NewS3File(localFile.FileName(), remoteDir)
	if err != nil {
		vm.errChan <- fmt.Errorf("error creating S3 file: %w", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	if err := vm.fileSvc.UploadFile(ctx, localFile, remoteFile); err != nil {
		vm.errChan <- fmt.Errorf("error uploading file: %w", err)
		return err
	}

	return vm.RefreshDir(remoteDir.ID)
}

func (vm *explorerViewModelImpl) DeleteFile(file *explorer.S3File) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	dir, err := vm.dirSvc.GetDirectoryByID(ctx, file.DirectoryID)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting parent directory: %w", err)
		return err
	}

	if err := vm.dirSvc.DeleteFile(ctx, dir, file.ID); err != nil {
		vm.errChan <- fmt.Errorf("error deleting file: %w", err)
		return err
	}

	if err := vm.tree.Remove(file.ID.String()); err != nil {
		vm.errChan <- fmt.Errorf("error removing file from tree: %w", err)
		return err
	}

	vm.mu.Lock()
	delete(vm.filesById, file.ID)
	vm.mu.Unlock()

	return nil
}

func (vm *explorerViewModelImpl) GetLastSaveDir() fyne.ListableURI {
	return vm.lastSaveDir
}

func (vm *explorerViewModelImpl) SetLastSaveDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SaveLastDir: %w", err)
	}
	vm.lastSaveDir = uriLister
	return nil
}

func (vm *explorerViewModelImpl) GetLastUploadDir() fyne.ListableURI {
	return vm.lastUploadDir
}

func (vm *explorerViewModelImpl) SetLastUploadDir(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SetLastUploadDir: %w", err)
	}
	vm.lastUploadDir = uriLister
	return nil
}

func (vm *explorerViewModelImpl) CreateEmptyDirectory(parent *explorer.S3Directory, name string) (*explorer.S3Directory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	subDir, err := vm.dirSvc.CreateSubDirectory(ctx, parent, name)
	if err != nil {
		vm.errChan <- fmt.Errorf("error creating subdirectory: %w", err)
		return nil, err
	}

	newNode := NewTreeNode(subDir.ID.String(), subDir.ID.ToName(), TreeNodeTypeDirectory)
	if err := vm.tree.Append(parent.ID.String(), subDir.ID.String(), newNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending new subdirectory to tree: %w", err)
		return nil, err
	}

	return subDir, nil
}

// func (vm *explorerViewModelImpl) IsReadOnly() bool {
// 	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
// 	defer cancel()
// 	currentConn, err := vm.connRepo.GetSelectedConnection(ctx)
// 	if err != nil {
// 		vm.errChan <- fmt.Errorf("error getting selected connection: %w", err)
// 		return false
// 	}
// 	return currentConn.ReadOnly
// }

func (vm *explorerViewModelImpl) resetTreeContent() {
	vm.tree = binding.NewUntypedTree()
}

func (vm *explorerViewModelImpl) initializeTreeData(ctx context.Context) error {
	rootDir, err := vm.dirSvc.GetRootDirectory(ctx)
	if err != nil {
		return fmt.Errorf("error getting root directory: %w", err)
	}
	currentConn, err := vm.connRepo.GetSelectedConnection(ctx)
	if err != nil {
		return fmt.Errorf("error getting selected connection: %w", err)
	}
	displayLabel := "Bucket: " + currentConn.BucketName
	rootNode := NewTreeNode(rootDir.ID.String(), displayLabel, TreeNodeTypeBucketRoot)
	if err := vm.tree.Append("", rootNode.ID, rootNode); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	if err := vm.OpenDirectory(rootDir.ID); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	return nil
}

func (vm *explorerViewModelImpl) fetchAndUpdateDirectory(ctx context.Context, dirID explorer.S3DirectoryID) (*explorer.S3Directory, error) {
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

func (vm *explorerViewModelImpl) removeDirectoryContent(dirID explorer.S3DirectoryID) error {
	oldDir, err := vm.GetDirByID(dirID)
	if err != nil {
		return nil // TODO: handle error properly
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

func (vm *explorerViewModelImpl) appendDirectoryContent(dirID explorer.S3DirectoryID, dir *explorer.S3Directory) error {
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

func (vm *explorerViewModelImpl) appendDirectoryNode(parentDirID explorer.S3DirectoryID, dirID explorer.S3DirectoryID) error {
	dirNode := NewTreeNode(dirID.String(), dirID.ToName(), TreeNodeTypeDirectory)
	dirNode.SetIsNotLoaded()

	if err := vm.tree.Append(parentDirID.String(), dirNode.ID, dirNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
		return err
	}

	return nil
}

func (vm *explorerViewModelImpl) appendFileNode(parentDirID explorer.S3DirectoryID, file *explorer.S3File) error {
	fileNode := NewTreeNode(file.ID.String(), file.Name, TreeNodeTypeFile)
	fileNode.SetIsLoaded()

	if err := vm.tree.Append(parentDirID.String(), fileNode.ID, fileNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
		return err
	}

	vm.mu.Lock()
	vm.filesById[file.ID] = file
	vm.mu.Unlock()

	return nil
}
