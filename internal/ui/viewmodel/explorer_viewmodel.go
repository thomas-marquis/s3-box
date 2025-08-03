package viewmodel

import (
	"context"
	"errors"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"

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
	ViewModel

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

	////////////////////////
	// Action methods
	////////////////////////

	// LoadDirectory sync a directory with the actual s3 one and load its files dans children.
	// If the directory is already open, it will do nothing.
	LoadDirectory(dirNode node.DirectoryNode) error // TODO: use this method for refreshing the content too

	// GetFileContent retrieves the content of the specified file, returning a Content object or an error if the operation fails.
	GetFileContent(f *directory.File) (*directory.Content, error)

	// DownloadFile downloads a file to the specified local destination
	DownloadFile(f *directory.File, dest string)

	// UploadFile uploads a local file to the specified remote directory
	UploadFile(localPath string, dir *directory.Directory)

	// DeleteFile removes a file from storage and updates the tree
	DeleteFile(file *directory.File)

	// UpdateLastDownloadLocation updates the last used save directory path
	UpdateLastDownloadLocation(filePath string) error

	// UpdateLastUploadLocation updates the last used upload directory path
	UpdateLastUploadLocation(filePath string)

	// CreateEmptyDirectory creates an empty subdirectory in the given parent directory
	CreateEmptyDirectory(parent *directory.Directory, name string)
}

type explorerViewModelImpl struct {
	baseViewModel

	mu                  sync.Mutex
	directoryRepository directory.Repository
	tree                binding.UntypedTree

	selectedConnection    binding.Untyped
	selectedConnectionVal *connection_deck.Connection

	settingsVm                SettingsViewModel
	lastDownloadLocation      fyne.ListableURI
	lastUploadDir             fyne.ListableURI
	displayNoConnectionBanner binding.Bool

	notifier notification.Repository
	bus      event.Bus
}

func NewExplorerViewModel(
	directoryRepository directory.Repository,
	settingsVm SettingsViewModel,
	notifier notification.Repository,
	initialConnection *connection_deck.Connection,
	bus event.Bus,
) ExplorerViewModel {
	vm := &explorerViewModelImpl{
		baseViewModel: baseViewModel{
			loading:      binding.NewBool(),
			errorMessage: binding.NewString(),
			infoMessage:  binding.NewString(),
		},
		settingsVm:            settingsVm,
		directoryRepository:   directoryRepository,
		notifier:              notifier,
		selectedConnectionVal: initialConnection,
		selectedConnection:    binding.NewUntyped(),
		bus:                   bus,
	}

	if err := vm.initializeTreeData(initialConnection); err != nil {
		if errors.Is(err, ErrNoConnectionSelected) {
			vm.selectedConnection.Set(nil)
			vm.selectedConnectionVal = nil
		}
		notifier.NotifyError(fmt.Errorf("error setting initial connection: %w", err))
	}

	go vm.listenEvents()

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

func (vm *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) {
	evt := f.Download(vm.selectedConnectionVal.ID(), dest)
	vm.bus.Publish(evt)
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory) {
	if vm.selectedConnectionVal == nil {
		err := vm.notifier.NotifyError(ErrNoConnectionSelected)
		vm.bus.Publish(directory.NewContentUploadedFailureEvent(err, dir))
		return
	}

	evt, err := dir.UploadFile(localPath)
	if err != nil {
		err := vm.notifier.NotifyError(fmt.Errorf("error uploading file: %w", err))
		vm.bus.Publish(directory.NewContentUploadedFailureEvent(err, dir))
		return
	}
	vm.bus.Publish(evt)
}

func (vm *explorerViewModelImpl) DeleteFile(file *directory.File) {
	dirNodeItem, err := vm.tree.GetValue(file.DirectoryPath().String())
	if err != nil {
		panic(
			fmt.Sprintf("impossible to retreive the direcotry you want to refresh: %s",
				file.DirectoryPath().String()))
	}

	dirNode, ok := dirNodeItem.(node.DirectoryNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", file.DirectoryPath().String()))
	}

	parent := dirNode.Directory()
	evt, err := parent.RemoveFile(file.Name())
	if err != nil {
		vm.bus.Publish(directory.NewFileDeletedFailureEvent(
			fmt.Errorf("error removing file from tthe direcory %s: %w", parent.Path(), err), parent))
		return
	}
	vm.bus.Publish(evt)
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

func (vm *explorerViewModelImpl) UpdateLastUploadLocation(filePath string) {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		vm.notifier.NotifyError(fmt.Errorf("update upload location: %w", err))
		return
	}
	vm.lastUploadDir = uriLister
}

func (vm *explorerViewModelImpl) CreateEmptyDirectory(parent *directory.Directory, name string) {
	if vm.selectedConnectionVal == nil {
		err := vm.notifier.NotifyError(ErrNoConnectionSelected)
		vm.bus.Publish(directory.NewCreatedFailureEvent(err, parent))
		return
	}

	evt, err := parent.NewSubDirectory(name)
	if err != nil {
		err := vm.notifier.NotifyError(fmt.Errorf("error creating subdirectory: %w", err))
		vm.bus.Publish(directory.NewCreatedFailureEvent(err, parent))
		return
	}

	vm.bus.Publish(evt)
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

func (vm *explorerViewModelImpl) addNewDirectoryToTree(dirToAdd *directory.Directory) error {
	parentPath := dirToAdd.Path().ParentPath()
	parentNodeItem, err := vm.tree.GetValue(parentPath.String())
	if err != nil {
		return fmt.Errorf("impossible to retreive the parent direcotry from path: %s", parentPath)
	}
	childNode := node.NewDirectoryNode(dirToAdd.Path())
	if err := vm.tree.Append(parentNodeItem.(node.DirectoryNode).ID(), childNode.ID(), childNode); err != nil {
		return fmt.Errorf("error appending directory to tree: %w", err)
	}
	return nil
}

func (vm *explorerViewModelImpl) addNewFileToTree(fileToAdd *directory.File) error {
	newFileNode := node.NewFileNode(fileToAdd)
	if err := vm.tree.Append(fileToAdd.DirectoryPath().String(), newFileNode.ID(), newFileNode); err != nil {
		return fmt.Errorf("error appending file to tree: %w", err)
	}
	return nil
}

func (vm *explorerViewModelImpl) resetSubTree(startNode node.DirectoryNode) error {
	return vm.tree.Remove(startNode.ID())
}

func (vm *explorerViewModelImpl) removeFileFromTree(file *directory.File) error {
	fileNodePath := file.FullPath()
	return vm.tree.Remove(fileNodePath)
}

func (vm *explorerViewModelImpl) listenEvents() {
	for evt := range vm.bus.Subscribe() {
		switch evt.Type() {
		case connection_deck.SelectEventType.AsSuccess():
			e := evt.(connection_deck.SelectSuccessEvent)
			conn := e.Connection()
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

		case connection_deck.RemoveEventType.AsSuccess():
			e := evt.(connection_deck.RemoveSuccessEvent)
			conn := e.Connection()
			if vm.selectedConnectionVal != nil && vm.selectedConnectionVal.Is(conn) {
				vm.selectedConnectionVal = nil
				vm.selectedConnection.Set(nil)
			}

		case directory.ContentUploadedEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case directory.ContentUploadedEventType.AsSuccess():
			e := evt.(directory.ContentUploadedSuccessEvent)
			if err := vm.addNewFileToTree(e.Content().File()); err != nil {
				vm.bus.Publish(directory.NewContentUploadedFailureEvent(err, e.Directory()))
				vm.loading.Set(false)
				continue
			}
			e.Directory().Notify(e)
			vm.loading.Set(false)

		case directory.ContentUploadedEventType.AsFailure():
			e := evt.(directory.ContentUploadedFailureEvent)
			e.Directory().Notify(e)
			err := vm.notifier.NotifyError(fmt.Errorf("error uploading file: %w", e.Error()))
			vm.errorMessage.Set(err.Error())
			vm.loading.Set(false)

		case directory.CreatedEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case directory.CreatedEventType.AsSuccess():
			e := evt.(directory.CreatedSuccessEvent)
			if err := vm.addNewDirectoryToTree(e.Directory()); err != nil {
				vm.bus.Publish(directory.NewCreatedFailureEvent(err, e.Parent()))
				vm.loading.Set(false)
				continue
			}
			e.Parent().Notify(e)
			vm.loading.Set(false)

		case directory.CreatedEventType.AsFailure():
			e := evt.(directory.CreatedFailureEvent)
			e.Parent().Notify(e)
			err := vm.notifier.NotifyError(fmt.Errorf("error creating directory: %w", e.Error()))
			vm.errorMessage.Set(err.Error())
			vm.loading.Set(false)

		case directory.FileDeletedEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case directory.FileDeletedEventType.AsSuccess():
			e := evt.(directory.FileDeletedSuccessEvent)

			if err := vm.removeFileFromTree(e.File()); err != nil {
				vm.bus.Publish(directory.NewFileDeletedFailureEvent(err, e.Parent()))
				vm.loading.Set(false)
				continue
			}
			e.Parent().Notify(e)
			vm.loading.Set(false)
			vm.infoMessage.Set(fmt.Sprintf("File %s deleted", e.File().Name()))

		case directory.FileDeletedEventType.AsFailure():
			e := evt.(directory.FileDeletedFailureEvent)
			e.Parent().Notify(e)
			err := vm.notifier.NotifyError(fmt.Errorf("error deleting file: %w", e.Error()))
			vm.errorMessage.Set(err.Error())
			vm.loading.Set(false)

		case directory.ContentDownloadEventType:
			if vm.IsLoading() {
				continue
			}
			vm.loading.Set(true)

		case directory.ContentDownloadEventType.AsSuccess():
			e := evt.(directory.ContentUploadedSuccessEvent)
			e.Directory().Notify(e)
			vm.loading.Set(false)
			vm.infoMessage.Set(fmt.Sprintf("File %s downloaded", e.Content().File().Name()))

		case directory.ContentDownloadEventType.AsFailure():
			e := evt.(directory.ContentUploadedFailureEvent)
			e.Directory().Notify(e)
			err := vm.notifier.NotifyError(fmt.Errorf("error downloading file: %w", e.Error()))
			vm.errorMessage.Set(err.Error())
			vm.loading.Set(false)
		}
	}
}
