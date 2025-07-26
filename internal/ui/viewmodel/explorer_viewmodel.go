package viewmodel

import (
	"context"
	"errors"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
)

// ExplorerViewModel represents the view model for the file explorer interface.
// It handles the tree structure display, file operations, and directory management
// while maintaining the connection with the underlying storage system.
type ExplorerViewModel interface {
	// OnDisplayNoConnectionBannerChange registers a callback function that is triggered
	// when the no-connection banner display state changes
	OnDisplayNoConnectionBannerChange(fn func(shouldDisplay bool))

	// ErrorChan returns a channel for error notifications from the view model
	ErrorChan() chan error

	// Tree returns the binding for the directory/file tree structure
	Tree() binding.UntypedTree

	// LoadDirectory sync a directory with the actual s3 one and load its files dans children.
	// If the directory is already open, it will do nothing.
	LoadDirectory(dirNode node.DirectoryNode) error // TODO: use this method for refreshing the content too

	//// RemoveDirToTree removes a directory and its contents from the tree structure
	//RemoveDirToTree(dirID directory.Path) error
	//
	//// GetDirByID retrieves a directory by its identifier from the cache
	//GetDirByID(dirID directory.Path) (*directory.Directory, error)
	//
	//// GetFileByName retrieves a file by its name from a specific parent directory
	//GetFileByName(parent directory.Path, name directory.FileName) (*directory.File, error)
	//
	//// PreviewFile returns the content of a file as a string for preview purposes
	//// Returns an error if the file is too large or cannot be read
	//PreviewFile(f *directory.File) (string, error)
	//
	//// ResetTree clears and reinitializes the entire tree structure
	//ResetTree() error

	// DownloadFile downloads a file to the specified local destination
	DownloadFile(f *directory.File, dest string) error

	// UploadFile uploads a local file to the specified remote directory
	UploadFile(localPath string, dir *directory.Directory) error

	//// DeleteFile removes a file from storage and updates the tree
	//DeleteFile(file *directory.File) error

	// LastDownloadLocation returns the URI of the last used save directory
	LastDownloadLocation() fyne.ListableURI

	// UpdateLastDownloadLocation updates the last used save directory path
	UpdateLastDownloadLocation(filePath string) error

	// LastUploadLocation returns the URI of the last used upload directory
	LastUploadLocation() fyne.ListableURI

	// UpdateLastUploadLocation updates the last used upload directory path
	UpdateLastUploadLocation(filePath string) error

	//// CreateEmptyDirectory creates an empty subdirectory in the given parent directory
	//CreateEmptyDirectory(parent *directory.Directory, name string) (*directory.Directory, error)
}

type explorerViewModelImpl struct {
	mu            sync.Mutex
	deck          *connection_deck.Deck
	connRepo      connection_deck.Repository
	dirRepository directory.Repository
	dirSvc        directory.Service
	tree          binding.UntypedTree
	errChan       chan error
	publisher     directory.EventPublisher

	settingsVm                SettingsViewModel
	lastDownloadLocation      fyne.ListableURI
	lastUploadDir             fyne.ListableURI
	displayNoConnectionBanner binding.Bool
}

var _ ExplorerViewModel = &explorerViewModelImpl{}

func NewExplorerViewModel(
	dirSvc directory.Service,
	connRepo connection_deck.Repository,
	dirRepo directory.Repository,
	settingsVm SettingsViewModel,
	publisher directory.EventPublisher,
) *explorerViewModelImpl {
	t := binding.NewUntypedTree()
	errChan := make(chan error)

	// Start error handler
	go func() {
		for err := range errChan {
			fmt.Printf("Error in ExplorerViewModel: %v\n", err)
		}
	}()

	vm := &explorerViewModelImpl{
		tree:                      t,
		dirSvc:                    dirSvc,
		settingsVm:                settingsVm,
		dirRepository:             dirRepo,
		errChan:                   errChan,
		connRepo:                  connRepo,
		displayNoConnectionBanner: binding.NewBool(),
		publisher:                 publisher,
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout()*2)
	defer cancel()

	deck, err := connRepo.Get(ctx)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting connection deck: %w", err)
		return vm
	}
	vm.deck = deck

	vm.displayNoConnectionBanner.Set(false)
	if err := vm.initializeTreeData(ctx); err != nil {
		if errors.Is(err, ErrNoConnectionSelected) {
			vm.displayNoConnectionBanner.Set(true)
		}
		vm.errChan <- fmt.Errorf("error resetting tree: %w", err)
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

func (vm *explorerViewModelImpl) Tree() binding.UntypedTree {
	return vm.tree
}

func (vm *explorerViewModelImpl) LoadDirectory(dirNode node.DirectoryNode) error {
	if dirNode.IsLoaded() {
		return nil
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, vm.settingsVm.CurrentTimeout())
	defer cancel()

	conn := vm.deck.SelectedConnection()

	dir, err := vm.dirRepository.GetByPath(ctx, conn.ID(), dirNode.Path())
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting directory: %w", err)
		return err
	}

	if err := dirNode.Load(dir); err != nil {
		vm.errChan <- fmt.Errorf("error loading directory: %w", err)
		return err
	}

	for _, file := range dir.Files() {
		fileNode := node.NewFileNode(file)
		if err := vm.tree.Append(dirNode.ID(), fileNode.ID(), fileNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
			continue
		}
	}

	for _, subDirPath := range dir.SubDirectories() {
		subDirNode := node.NewDirectoryNode(subDirPath)
		if err := vm.tree.Append(dirNode.ID(), subDirNode.ID(), subDirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
			continue
		}
	}

	return nil
}

//	func (vm *explorerViewModelImpl) RemoveDirToTree(dirID explorer.S3DirectoryID) error {
//		item, err := vm.tree.GetValue(dirID.String())
//		if err != nil {
//			return fmt.Errorf("impossible to retreive the direcotry you want to remove: %s", dirID.String())
//		}
//		node, ok := item.(*TreeNode)
//		if !ok {
//			panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirID.String()))
//		}
//
//		if !node.IsLoaded() {
//			if err := vm.tree.Remove(dirID.String()); err != nil {
//				vm.errChan <- fmt.Errorf("error removing directory from tree: %w", err)
//				return err
//			}
//		} else {
//			vm.mu.Lock()
//			dir, ok := vm.dirsById[dirID]
//			vm.mu.Unlock()
//			if !ok {
//				panic(fmt.Sprintf("impossible to find the directory in the cache: %s", dirID.String()))
//			}
//			for _, f := range dir.Files {
//				if err := vm.tree.Remove(f.ID.String()); err != nil {
//					vm.errChan <- fmt.Errorf("error removing file from tree: %w", err)
//					return err
//				}
//			}
//			for _, subDirID := range dir.SubDirectoriesIDs {
//				if err := vm.RemoveDirToTree(subDirID); err != nil {
//					vm.errChan <- fmt.Errorf("error removing subdirectory from tree: %w", err)
//					return err
//				}
//			}
//		}
//
//		return nil
//	}
//
//	func (vm *explorerViewModelImpl) GetDirByID(dirID explorer.S3DirectoryID) (*explorer.S3Directory, error) {
//		vm.mu.Lock()
//		dir, ok := vm.dirsById[dirID]
//		vm.mu.Unlock()
//		if !ok {
//			return nil, fmt.Errorf("directory not found in cache")
//		}
//		return dir, nil
//	}
//
//	func (vm *explorerViewModelImpl) GetFileByName(parent directory.Path, name directory.FileName) (*directory.File, error) {
//		vm.mu.Lock()
//		file, ok := vm.filesById[fileID]
//		vm.mu.Unlock()
//		if !ok {
//			return nil, fmt.Errorf("file not found in cache")
//		}
//		return file, nil
//	}
//
//	func (vm *explorerViewModelImpl) PreviewFile(f *explorer.S3File) (string, error) {
//		if f.SizeBytes > vm.GetMaxFileSizePreview() {
//			return "", fmt.Errorf("file is too big to PreviewFile")
//		}
//
//		ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
//		defer cancel()
//		content, err := vm.fileSvc.GetContent(ctx, f)
//		if err != nil {
//			vm.errChan <- fmt.Errorf("error getting file content: %w", err)
//			return "", err
//		}
//
//		return string(content), nil
//	}
//
//	func (vm *explorerViewModelImpl) ResetTree() error {
//		vm.resetTreeContent()
//		return vm.initializeTreeData(context.Background())
//	}

func (vm *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	evt := f.Download(vm.deck.SelectedConnection().ID(), dest)
	evt.AttachContext(ctx)
	vm.publisher.Publish(evt)

	return nil
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory) error {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()
	evt, err := dir.UploadFile(localPath)
	if err != nil {
		vm.errChan <- fmt.Errorf("error uploading file: %w", err)
		return err
	}
	evt.AttachContext(ctx)
	vm.publisher.Publish(evt)

	return vm.RefreshDir(remoteDir.ID)
}

//	func (vm *explorerViewModelImpl) DeleteFile(file *explorer.S3File) error {
//		ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
//		defer cancel()
//
//		dir, err := vm.dirSvc.GetDirectoryByID(ctx, file.DirectoryID)
//		if err != nil {
//			vm.errChan <- fmt.Errorf("error getting parent directory: %w", err)
//			return err
//		}
//
//		if err := vm.dirSvc.DeleteFile(ctx, dir, file.ID); err != nil {
//			vm.errChan <- fmt.Errorf("error deleting file: %w", err)
//			return err
//		}
//
//		if err := vm.tree.Remove(file.ID.String()); err != nil {
//			vm.errChan <- fmt.Errorf("error removing file from tree: %w", err)
//			return err
//		}
//
//		vm.mu.Lock()
//		delete(vm.filesById, file.ID)
//		vm.mu.Unlock()
//
//		return nil
//	}

func (vm *explorerViewModelImpl) LastDownloadLocation() fyne.ListableURI {
	return vm.lastDownloadLocation
}

func (vm *explorerViewModelImpl) UpdateLastDownloadLocation(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("SaveLastDir: %w", err)
	}
	vm.lastDownloadLocation = uriLister
	return nil
}

func (vm *explorerViewModelImpl) LastUploadLocation() fyne.ListableURI {
	return vm.lastUploadDir
}

func (vm *explorerViewModelImpl) UpdateLastUploadLocation(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		return fmt.Errorf("UpdateLastUploadLocation: %w", err)
	}
	vm.lastUploadDir = uriLister
	return nil
}

//	func (vm *explorerViewModelImpl) CreateEmptyDirectory(parent *explorer.S3Directory, name string) (*explorer.S3Directory, error) {
//		ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
//		defer cancel()
//
//		subDir, err := vm.dirSvc.CreateSubDirectory(ctx, parent, name)
//		if err != nil {
//			vm.errChan <- fmt.Errorf("error creating subdirectory: %w", err)
//			return nil, err
//		}
//
//		newNode := NewTreeNode(subDir.ID.String(), subDir.ID.ToName(), TreeNodeTypeDirectory)
//		if err := vm.tree.Append(parent.ID.String(), subDir.ID.String(), newNode); err != nil {
//			vm.errChan <- fmt.Errorf("error appending new subdirectory to tree: %w", err)
//			return nil, err
//		}
//
//		return subDir, nil
//	}
//
//	func (vm *explorerViewModelImpl) resetTreeContent() {
//		vm.tree = binding.NewUntypedTree()
//	}
func (vm *explorerViewModelImpl) initializeTreeData(ctx context.Context) error {
	currentConn := vm.deck.SelectedConnection()
	if currentConn == nil {
		return ErrNoConnectionSelected
	}

	displayLabel := "Bucket: " + currentConn.Bucket()

	rootNode := node.NewDirectoryNode(directory.RootPath, node.WithDisplayName(displayLabel))
	if err := vm.tree.Append("", rootNode.ID(), rootNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending directory to tree: %w", err)
		return err
	}

	if err := vm.LoadDirectory(rootNode); err != nil {
		return fmt.Errorf("error appending root directory to tree: %w", err)
	}

	return nil
}

func (vm *explorerViewModelImpl) sync(dir *directory.Directory) error {
	dirTreeNodeItem, err := vm.tree.GetValue(dir.Path().String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the direcotry you want to refresh: %s", dirPath.String())
	}
	dirTreeNode, ok := dirTreeNodeItem.(*node.DirectoryNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dirPath.String()))
	}

	if err := vm.removeDirectoryContent(dir); err != nil {
		return err
	}

	return vm.appendDirectoryContent(dir)
}

//	func (vm *explorerViewModelImpl) fetchAndUpdateDirectory(ctx context.Context, dirID explorer.S3DirectoryID) (*explorer.S3Directory, error) {
//		dir, err := vm.dirSvc.GetDirectoryByID(ctx, dirID)
//		if err != nil {
//			vm.errChan <- fmt.Errorf("error getting directory by ID (%s): %w", dirID, err)
//			return nil, err
//		}
//
//		vm.mu.Lock()
//		vm.dirsById[dirID] = dir
//		vm.mu.Unlock()
//
//		return dir, nil
//	}
func (vm *explorerViewModelImpl) removeDirectoryContent(dir *directory.Directory) error {
	for _, file := range dir.Files() {
		if err := vm.tree.Remove(file.FullPath()); err != nil {
			vm.errChan <- fmt.Errorf("error removing old file from tree: %w", err)
		}
	}

	for _, subDirPath := range dir.SubDirectories() {
		if err := vm.tree.Remove(subDirPath.String()); err != nil {
			vm.errChan <- fmt.Errorf("error removing old subdirectory from tree: %w", err)
		}
	}

	return nil
}

func (vm *explorerViewModelImpl) appendDirectoryContent(dir *directory.Directory) error {
	for _, file := range dir.Files() {
		if err := vm.appendFileNode(dir, file); err != nil {
			continue
		}
	}

	for _, subDirID := range dir.SubDirectories() {
		if err := vm.appendDirectoryNode(dirID, subDirID); err != nil {
			continue
		}
	}

	return nil
}

func (vm *explorerViewModelImpl) appendDirectoryNode(dir *directory.Directory) error {
	dirNode := node.NewDirectoryNode(dir.Path())

	if err := vm.tree.Append(parentDirID.String(), dirNode.ID, dirNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
		return err
	}

	return nil
}

func (vm *explorerViewModelImpl) appendFileNode(parent *directory.Directory, file *directory.File) error {
	fileNode := node.NewFileNode(file)

	if err := vm.tree.Append(parent.Path().String(), fileNode.ID(), fileNode); err != nil {
		vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
		return err
	}

	return nil
}
