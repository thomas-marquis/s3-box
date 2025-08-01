package viewmodel

import (
	"context"
	"errors"

	"fmt"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
	"github.com/thomas-marquis/s3-box/internal/ui/uievent"
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
	////////////////////////
	// State methods
	////////////////////////

	// Tree returns the binding for the directory/file tree structure
	Tree() binding.UntypedTree

	SelectedConnection() binding.Untyped

	CurrentSelectedConnection() *connection_deck.Connection

	// LastDownloadLocation returns the URI of the last used save directory
	LastDownloadLocation() fyne.ListableURI

	// LastUploadLocation returns the URI of the last used upload directory
	LastUploadLocation() fyne.ListableURI

	Loading() binding.Bool

	////////////////////////
	// Action methods
	////////////////////////

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

	// UpdateLastDownloadLocation updates the last used save directory path
	UpdateLastDownloadLocation(filePath string) error

	// UpdateLastUploadLocation updates the last used upload directory path
	UpdateLastUploadLocation(filePath string) error

	// CreateEmptyDirectory creates an empty subdirectory in the given parent directory
	CreateEmptyDirectory(parent *directory.Directory, name string) (*directory.Directory, error)
}

type explorerViewModelImpl struct {
	mu                  sync.Mutex
	directoryRepository directory.Repository
	tree                binding.UntypedTree
	domainPublisher     *directory.EventPublisher

	selectedConnection    binding.Untyped
	selectedConnectionVal *connection_deck.Connection

	loading binding.Bool

	settingsVm                SettingsViewModel
	lastDownloadLocation      fyne.ListableURI
	lastUploadDir             fyne.ListableURI
	displayNoConnectionBanner binding.Bool

	uiPublisher uievent.Publisher
	notifier    notification.Repository
}

func NewExplorerViewModel(
	directoryRepository directory.Repository,
	settingsVm SettingsViewModel,
	domainPublisher *directory.EventPublisher,
	uiPublisher uievent.Publisher,
	notifier notification.Repository,
	initialConnection *connection_deck.Connection,
) ExplorerViewModel {
	vm := &explorerViewModelImpl{
		settingsVm:            settingsVm,
		directoryRepository:   directoryRepository,
		domainPublisher:       domainPublisher,
		notifier:              notifier,
		uiPublisher:           uiPublisher,
		selectedConnectionVal: initialConnection,
		selectedConnection:    binding.NewUntyped(),
		loading:               binding.NewBool(),
	}

	if err := vm.initializeTreeData(initialConnection); err != nil {
		if errors.Is(err, ErrNoConnectionSelected) {
			vm.selectedConnection.Set(nil)
			vm.selectedConnectionVal = nil
		}
		notifier.NotifyError(fmt.Errorf("error setting initial connection: %w", err))
	}

	go vm.listenUiEvents()

	return vm
}

func (vm *explorerViewModelImpl) Tree() binding.UntypedTree {
	return vm.tree
}

func (vm *explorerViewModelImpl) SelectedConnection() binding.Untyped {
	return vm.selectedConnection
}

func (vm *explorerViewModelImpl) CurrentSelectedConnection() *connection_deck.Connection {
	return vm.selectedConnectionVal
}

func (vm *explorerViewModelImpl) Loading() binding.Bool {
	return vm.loading
}

func (vm *explorerViewModelImpl) LoadDirectory(dirNode node.DirectoryNode) error {
	if vm.selectedConnectionVal == nil {
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
	if vm.selectedConnectionVal == nil {
		return nil, vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	if file.SizeBytes() > vm.settingsVm.CurrentMaxFilePreviewSizeBytes() {
		return nil, vm.notifier.NotifyError(fmt.Errorf("file is too big to GetFileContent"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	content, err := vm.directoryRepository.GetFileContent(ctx, vm.selectedConnectionVal.ID(), file)
	if err != nil {
		return nil, vm.notifier.NotifyError(fmt.Errorf("error getting file content: %w", err))
	}

	return content, nil
}

func (vm *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) error {
	if vm.selectedConnectionVal == nil {
		return vm.notifier.NotifyError(ErrNoConnectionSelected)
	}
	evt := f.Download(vm.selectedConnectionVal.ID(), dest)
	vm.domainPublisher.Publish(evt)
	return nil
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory) error {
	if vm.selectedConnectionVal == nil {
		return vm.notifier.NotifyError(ErrNoConnectionSelected)
	}

	evt, err := dir.UploadFile(localPath)
	if err != nil {
		return vm.notifier.NotifyError(fmt.Errorf("error uploading file: %w", err))
	}
	vm.domainPublisher.Publish(evt)
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
	vm.domainPublisher.Publish(evt)

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
	if vm.selectedConnectionVal == nil {
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
	vm.domainPublisher.Publish(evt)
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

	dir, err := vm.directoryRepository.GetByPath(ctx, vm.selectedConnectionVal.ID(), dirID)
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

func (vm *explorerViewModelImpl) listenUiEvents() {
	for event := range vm.uiPublisher.Subscribe() {
		switch event.Type() {
		case uievent.SelectConnectionSuccessType:
			evt := event.(*uievent.SelectConnectionSuccess)
			conn := evt.Connection
			hasChanged := (vm.selectedConnectionVal == nil && conn != nil) ||
				(vm.selectedConnectionVal != nil && conn == nil) ||
				(vm.selectedConnectionVal != nil && !vm.selectedConnectionVal.Is(conn))
			if hasChanged {
				vm.loading.Set(true)
				vm.selectedConnectionVal = conn
				vm.selectedConnection.Set(conn)
				vm.initializeTreeData(conn)
				vm.loading.Set(false)
			}

		case uievent.DeleteConnectionSuccessType:
			evt := event.(*uievent.SelectConnectionSuccess)
			conn := evt.Connection
			if vm.selectedConnectionVal != nil && vm.selectedConnectionVal.Is(conn) {
				vm.selectedConnectionVal = nil
				vm.selectedConnection.Set(nil)
			}
		}

	}
}
