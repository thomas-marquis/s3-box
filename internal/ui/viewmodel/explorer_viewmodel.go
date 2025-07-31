package viewmodel

import (
	"context"
	"errors"
	"fmt"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
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

	// Tree returns the binding for the directory/file tree structure
	Tree() binding.UntypedTree

	// LoadDirectory sync a directory with the actual s3 one and load its files dans children.
	// If the directory is already open, it will do nothing.
	LoadDirectory(dirNode node.DirectoryNode) error // TODO: use this method for refreshing the content too

	// GetFileContent retrieves the content of the specified file, returning a Content object or an error if the operation fails.
	GetFileContent(f *directory.File) (*directory.Content, error)

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
	mu                  sync.Mutex
	connectionViewModel ConnectionViewModel
	directoryRepository directory.Repository
	tree                binding.UntypedTree
	publisher           *directory.EventPublisher
	selectedConnection  *connection_deck.Connection

	settingsVm                SettingsViewModel
	lastDownloadLocation      fyne.ListableURI
	lastUploadDir             fyne.ListableURI
	displayNoConnectionBanner binding.Bool

	notifier notification.Repository
}

func NewExplorerViewModel(
	connectionViewModel ConnectionViewModel,
	dirRepo directory.Repository,
	settingsVm SettingsViewModel,
	publisher *directory.EventPublisher,
	notifier notification.Repository,
) ExplorerViewModel {
	vm := &explorerViewModelImpl{
		settingsVm:                settingsVm,
		directoryRepository:       dirRepo,
		displayNoConnectionBanner: binding.NewBool(),
		publisher:                 publisher,
		notifier:                  notifier,
		connectionViewModel:       connectionViewModel,
	}

	//connectionViewModel.OnSelectedConnectionChanged(func(c *connection_deck.Connection) {
	//	if c == nil {
	//		vm.displayNoConnectionBanner.Set(true)
	//		return
	//	}
	//
	//	vm.displayNoConnectionBanner.Set(false)
	//	if !c.Is(vm.selectedConnection) {
	//		if err := vm.initializeTreeData(c); err != nil {
	//			notifier.NotifyError(fmt.Errorf("error resetting tree: %w", err))
	//			return
	//		}
	//		vm.selectedConnection = c
	//	}
	//})

	vm.selectedConnection = connectionViewModel.Deck().SelectedConnection()
	vm.displayNoConnectionBanner.Set(false)
	if err := vm.initializeTreeData(vm.selectedConnection); err != nil {
		if errors.Is(err, ErrNoConnectionSelected) {
			vm.displayNoConnectionBanner.Set(true)
		}
		notifier.NotifyError(fmt.Errorf("error resetting tree: %w", err))
	}

	return vm
}

func (vm *explorerViewModelImpl) OnDisplayNoConnectionBannerChange(fn func(shouldDisplay bool)) {
	vm.displayNoConnectionBanner.AddListener(binding.NewDataListener(func() {
		shouldDisplay, _ := vm.displayNoConnectionBanner.Get()
		fn(shouldDisplay)
	}))
}

func (vm *explorerViewModelImpl) Tree() binding.UntypedTree {
	return vm.tree
}

func (vm *explorerViewModelImpl) LoadDirectory(dirNode node.DirectoryNode) error {
	if vm.selectedConnection == nil {
		return vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	if dirNode.IsLoaded() {
		return nil
	}

	dir, err := vm.fetchDirectory(dirNode.Path())
	if err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error getting directory: %w", err))
	}

	if err := dirNode.Load(dir); err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error loading directory: %w", err))
	}

	if err := vm.fillSubTree(dirNode, dir); err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error filling sub tree: %w", err))
	}

	return nil
}

func (vm *explorerViewModelImpl) GetFileContent(file *directory.File) (*directory.Content, error) {
	if vm.selectedConnection == nil {
		return nil, vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	if file.SizeBytes() > vm.settingsVm.CurrentMaxFilePreviewSizeBytes() {
		return nil, vm.notifier.NotifyError(fmt.Errorf("file is too big to GetFileContent"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	content, err := vm.directoryRepository.GetFileContent(ctx, vm.selectedConnection.ID(), file)
	if err != nil {
		return nil, vm.notifier.NotifyError(fmt.Errorf("error getting file content: %w", err))
	}

	return content, nil
}

func (vm *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) error {
	if vm.selectedConnection == nil {
		return vm.notifier.NotifyError(ErrNoConnectionSelected)
	}
	evt := f.Download(vm.selectedConnection.ID(), dest)
	vm.publisher.Publish(evt)
	return nil
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory) error {
	if vm.selectedConnection == nil {
		return vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	evt, err := dir.UploadFile(localPath)
	if err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error uploading file: %w", err))
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
		return vm.notifier.NotifyError(fmt.Errorf("error removing file: %w", err))
	}
	evt.AttachErrorCallback(func(err error) {
		vm.notifier.NotifyError(fmt.Errorf("error removing file: %w", err))
	})
	evt.AttachSuccessCallback(func() {
		if err := vm.tree.Remove(file.FullPath()); err != nil {
			vm.notifier.NotifyError(fmt.Errorf("error removing file from tree: %w", err))
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
		return vm.notifier.NotifyError(fmt.Errorf("update download location: %w", err))
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
		return vm.notifier.NotifyError(fmt.Errorf("update upload location: %w", err))
	}
	vm.lastUploadDir = uriLister
	return nil
}

func (vm *explorerViewModelImpl) CreateEmptyDirectory(parent *directory.Directory, name string) (*directory.Directory, error) {
	if vm.selectedConnection == nil {
		return nil, vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	evt, err := parent.NewSubDirectory(name)
	if err != nil {
		return nil, vm.notifier.NotifyError(fmt.Errorf("error creating subdirectory: %w", err))
	}
	evt.AttachSuccessCallback(func() {
		if err := vm.sync(parent); err != nil {
			vm.notifier.NotifyError(fmt.Errorf("error syncing tree for the new directory: %w", err))
		}
	})
	evt.AttachErrorCallback(func(err error) {
		vm.notifier.NotifyError(fmt.Errorf("error creating subdirectory: %w", err))
	})
	vm.publisher.Publish(evt)
	return nil, nil
}

func (vm *explorerViewModelImpl) initializeTreeData(c *connection_deck.Connection) error {
	vm.tree = binding.NewUntypedTree()

	if c == nil {
		return vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	displayLabel := "Bucket: " + c.Bucket()

	rootNode := node.NewDirectoryNode(directory.RootPath, node.WithDisplayName(displayLabel))
	if err := vm.tree.Append("", rootNode.ID(), rootNode); err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error appending directory to tree: %w", err))
	}

	if err := vm.LoadDirectory(rootNode); err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error loading root directory: %w", err))
	}

	return nil
}

func (vm *explorerViewModelImpl) sync(dir *directory.Directory) error {
	dirNodeItem, err := vm.tree.GetValue(dir.Path().String())
	if err != nil {
		return vm.notifier.NotifyError(
			fmt.Errorf("impossible to retreive the direcotry you want to refresh: %s", dir.Path().String()))
	}
	dirNode, ok := dirNodeItem.(node.DirectoryNode)
	if !ok {
		return vm.notifier.NotifyError(
			fmt.Errorf("impossible to cast the item to TreeNode: %s", dir.Path().String()))
	}

	if !dirNode.IsLoaded() {
		if err := vm.LoadDirectory(dirNode); err != nil { // TODO: is a good idea forcing to load the dir here??
			return vm.notifier.NotifyError(fmt.Errorf("error loading directory: %w", err))
		}
		return nil
	}

	moreRecentDir, err := vm.fetchDirectory(dir.Path())
	if err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error getting directory: %w", err))
	}

	if moreRecentDir.Equal(dir) {
		return nil
	}

	if err := vm.tree.Remove(dirNode.ID()); err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error removing directory from tree: %w", err))
	}

	if err := vm.fillSubTree(dirNode, moreRecentDir); err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error filling sub tree: %w", err))
	}

	return nil
}

func (vm *explorerViewModelImpl) fetchDirectory(dirID directory.Path) (*directory.Directory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	dir, err := vm.directoryRepository.GetByPath(ctx, vm.selectedConnection.ID(), dirID)
	if err != nil {
		return nil, vm.notifier.NotifyError(fmt.Errorf("error getting directory: %w", err))
	}

	return dir, nil
}

func (vm *explorerViewModelImpl) fillSubTree(startNode node.DirectoryNode, dir *directory.Directory) error {
	for _, file := range dir.Files() {
		fileNode := node.NewFileNode(file)
		if err := vm.tree.Append(startNode.ID(), fileNode.ID(), fileNode); err != nil {
			vm.notifier.NotifyError(fmt.Errorf("error appending file to tree: %w", err))
			continue
		}
	}

	for _, subDirPath := range dir.SubDirectories() {
		subDirNode := node.NewDirectoryNode(subDirPath)
		if err := vm.tree.Append(startNode.ID(), subDirNode.ID(), subDirNode); err != nil {
			vm.notifier.NotifyError(fmt.Errorf("error appending subdirectory to tree: %w", err))
			continue
		}
	}
	return nil
}
