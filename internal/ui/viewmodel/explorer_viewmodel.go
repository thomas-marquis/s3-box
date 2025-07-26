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

	// DeleteFile removes a file from storage and updates the tree
	DeleteFile(file *directory.File) error

	// LastDownloadLocation returns the URI of the last used save directory
	LastDownloadLocation() fyne.ListableURI

	// UpdateLastDownloadLocation updates the last used save directory path
	UpdateLastDownloadLocation(filePath string) error

	// LastUploadLocation returns the URI of the last used upload directory
	LastUploadLocation() fyne.ListableURI

	// UpdateLastUploadLocation updates the last used upload directory path
	UpdateLastUploadLocation(filePath string) error

	// CreateEmptyDirectory creates an empty subdirectory in the given parent directory
	CreateEmptyDirectory(parent *directory.Directory, name string) (*directory.Directory, error)
}

type explorerViewModelImpl struct {
	mu            sync.Mutex
	deck          *connection_deck.Deck
	connRepo      connection_deck.Repository
	dirRepository directory.Repository
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

	dir, err := vm.fetchDirectory(dirNode.Path())
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting directory: %w", err)
		return err
	}

	if err := dirNode.Load(dir); err != nil {
		vm.errChan <- fmt.Errorf("error loading directory: %w", err)
		return err
	}

	if err := vm.fillSubTree(dirNode, dir); err != nil {
		vm.errChan <- fmt.Errorf("error filling sub tree: %w", err)
		return err
	}

	return nil
}

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
//		vm.tree = binding.NewUntypedTree()
//		return vm.initializeTreeData(context.Background())
//	}

func (vm *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) error {
	evt := f.Download(vm.deck.SelectedConnection().ID(), dest)
	vm.publisher.Publish(evt)
	return nil
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory) error {
	evt, err := dir.UploadFile(localPath)
	if err != nil {
		vm.errChan <- fmt.Errorf("error uploading file: %w", err)
		return err
	}
	vm.publisher.Publish(evt)
	return vm.sync(dir)
}

func (vm *explorerViewModelImpl) DeleteFile(file *directory.File) error {
	dirNodeItem, err := vm.tree.GetValue(file.DirectoryPath().String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the direcotry you want to remove: %s", file.DirectoryPath().String())
	}
	dirNode, ok := dirNodeItem.(node.DirectoryNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", file.DirectoryPath().String()))
	}

	dir := dirNode.Directory()
	evt, err := dir.RemoveFile(file.Name())
	if err != nil {
		vm.errChan <- fmt.Errorf("error removing file: %w", err)
		return err
	}
	evt.AttachErrorCallback(func(err error) {
		vm.errChan <- fmt.Errorf("error removing file: %w", err)
	})
	evt.AttachSuccessCallback(func() {
		if err := vm.tree.Remove(file.FullPath()); err != nil {
			vm.errChan <- fmt.Errorf("error removing file from tree: %w", err)
		}
	})
	vm.publisher.Publish(evt)

	return nil
}

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

func (vm *explorerViewModelImpl) CreateEmptyDirectory(parent *directory.Directory, name string) (*directory.Directory, error) {

	evt, err := parent.NewSubDirectory(name)
	if err != nil {
		vm.errChan <- fmt.Errorf("error creating subdirectory: %w", err)
		return nil, err
	}
	evt.AttachSuccessCallback(func() {
		if err := vm.sync(parent); err != nil {
			vm.errChan <- fmt.Errorf("error syncing tree for the new directory: %w", err)
		}
	})
	evt.AttachErrorCallback(func(err error) {
		vm.errChan <- fmt.Errorf("error creating subdirectory: %w", err)
	})
	vm.publisher.Publish(evt)
	return nil, nil
}

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
	dirNodeItem, err := vm.tree.GetValue(dir.Path().String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the direcotry you want to refresh: %s", dir.Path().String())
	}
	dirNode, ok := dirNodeItem.(node.DirectoryNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", dir.Path().String()))
	}

	if !dirNode.IsLoaded() {
		return vm.LoadDirectory(dirNode) // TODO: is a good idea forcing to load the dir here??
	}

	moreRecentDir, err := vm.fetchDirectory(dir.Path())
	if err != nil {
		return err
	}

	if moreRecentDir.Equal(dir) {
		return nil
	}

	if err := vm.tree.Remove(dirNode.ID()); err != nil {
		return err
	}

	if err := vm.fillSubTree(dirNode, moreRecentDir); err != nil {
		return err
	}

	return nil
}

func (vm *explorerViewModelImpl) fetchDirectory(dirID directory.Path) (*directory.Directory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	dir, err := vm.dirRepository.GetByPath(ctx, vm.deck.SelectedConnection().ID(), dirID)
	if err != nil {
		vm.errChan <- fmt.Errorf("error getting directory: %w", err)
		return nil, err
	}

	return dir, nil
}

func (vm *explorerViewModelImpl) fillSubTree(startNode node.DirectoryNode, dir *directory.Directory) error {
	for _, file := range dir.Files() {
		fileNode := node.NewFileNode(file)
		if err := vm.tree.Append(startNode.ID(), fileNode.ID(), fileNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending file to tree: %w", err)
			continue
		}
	}

	for _, subDirPath := range dir.SubDirectories() {
		subDirNode := node.NewDirectoryNode(subDirPath)
		if err := vm.tree.Append(startNode.ID(), subDirNode.ID(), subDirNode); err != nil {
			vm.errChan <- fmt.Errorf("error appending subdirectory to tree: %w", err)
			continue
		}
	}
	return nil
}
