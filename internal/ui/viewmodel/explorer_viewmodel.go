package viewmodel

import (
	"context"
	"errors"

	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"

	"fmt"
	"path/filepath"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/ui/node"

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
	Tree() binding.Tree[node.Node]

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
	UploadFile(localPath string, dir *directory.Directory, overwrite bool) error

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

	directoryRepository directory.Repository
	tree                binding.Tree[node.Node]

	selectedConnection    binding.Untyped
	selectedConnectionVal *connection_deck.Connection

	settingsVm           SettingsViewModel
	lastDownloadLocation fyne.ListableURI
	lastUploadDir        fyne.ListableURI

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
			vm.selectedConnection.Set(nil) //nolint:errcheck
			vm.selectedConnectionVal = nil
		}
		notifier.NotifyError(fmt.Errorf("error setting initial connection: %w", err))
	}

	go vm.listenEvents()

	return vm
}

func (vm *explorerViewModelImpl) Tree() binding.Tree[node.Node] {
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
		err := ErrNoConnectionSelected
		vm.notifier.NotifyError(err)
		return err
	}

	if dirNode.Directory().IsLoaded() {
		return nil
	}

	evt, err := dirNode.Directory().Load()
	if err != nil {
		wErr := fmt.Errorf("error loading directory: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
	}
	vm.bus.Publish(evt)

	return nil
}

func (vm *explorerViewModelImpl) GetFileContent(file *directory.File) (*directory.Content, error) {
	if vm.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		vm.notifier.NotifyError(err)
		return nil, err
	}

	if file.SizeBytes() > vm.settingsVm.CurrentMaxFilePreviewSizeBytes() {
		err := fmt.Errorf("file is too big to GetFileContent")
		vm.notifier.NotifyError(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), vm.settingsVm.CurrentTimeout())
	defer cancel()

	content, err := vm.directoryRepository.GetFileContent(ctx, vm.selectedConnectionVal.ID(), file)
	if err != nil {
		newErr := fmt.Errorf("error getting file content: %w", err)
		vm.notifier.NotifyError(newErr)
		return nil, newErr
	}

	return content, nil
}

func (vm *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) {
	evt := f.Download(vm.selectedConnectionVal.ID(), dest)
	vm.bus.Publish(evt)
}

func (vm *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory, overwrite bool) error {
	if vm.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		vm.notifier.NotifyError(err)
		vm.bus.Publish(directory.NewContentUploadedFailureEvent(err, dir))
		return nil
	}

	evt, err := dir.UploadFile(localPath, overwrite)
	if err != nil {
		if errors.Is(err, directory.ErrAlreadyExists) {
			return err
		}
		err := fmt.Errorf("error uploading file: %w", err)
		vm.notifier.NotifyError(err)
		vm.bus.Publish(directory.NewContentUploadedFailureEvent(err, dir))
		return nil
	}
	vm.bus.Publish(evt)
	return nil
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
		wErr := fmt.Errorf("update download location: %w", err)
		vm.notifier.NotifyError(wErr)
		return wErr
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
		err := ErrNoConnectionSelected
		vm.notifier.NotifyError(err)
		vm.bus.Publish(directory.NewCreatedFailureEvent(err, parent))
		return
	}

	evt, err := parent.NewSubDirectory(name)
	if err != nil {
		wErr := fmt.Errorf("error creating subdirectory: %w", err)
		vm.notifier.NotifyError(wErr)
		vm.bus.Publish(directory.NewCreatedFailureEvent(wErr, parent))
		return
	}

	vm.bus.Publish(evt)
}

func (vm *explorerViewModelImpl) initializeTreeData(c *connection_deck.Connection) error {
	vm.tree = binding.NewTree[node.Node](func(n1 node.Node, n2 node.Node) bool {
		return n1.ID() == n2.ID()
	})

	if c == nil {
		err := ErrNoConnectionSelected
		vm.notifier.NotifyError(err)
		return err
	}

	displayLabel := "Bucket: " + c.Bucket()

	rootDir, err := directory.New(c.ID(), directory.RootDirName, directory.NilParentPath)
	if err != nil {
		newErr := fmt.Errorf("error initializing the root directory: %w", err)
		vm.notifier.NotifyError(newErr)
		return newErr
	}
	rootNode := node.NewDirectoryNode(rootDir, node.WithDisplayName(displayLabel))
	if err := vm.tree.Append("", rootNode.ID(), rootNode); err != nil {
		newErr := fmt.Errorf("error appending directory to tree: %w", err)
		vm.notifier.NotifyError(newErr)
		return newErr
	}

	if err := vm.LoadDirectory(rootNode); err != nil {
		newErr := fmt.Errorf("error loading root directory: %w", err)
		vm.notifier.NotifyError(newErr)
		return newErr
	}

	return nil
}

func (vm *explorerViewModelImpl) fillSubTree(dir *directory.Directory) error {
	files, err := dir.Files()
	if err != nil {
		vm.notifier.NotifyError(fmt.Errorf("error getting files: %w", err))
		return err
	}

	subDirs, err := dir.SubDirectories()
	if err != nil {
		vm.notifier.NotifyError(fmt.Errorf("error getting subdirectories: %w", err))
		return err
	}

	for _, file := range files {
		fileNode := node.NewFileNode(file)
		if err := vm.tree.Append(dir.Path().String(), fileNode.ID(), fileNode); err != nil {
			vm.notifier.NotifyError(fmt.Errorf("error appending file to tree: %w", err))
			continue
		}
	}

	for _, subDirPath := range subDirs {
		subDirNode := node.NewDirectoryNode(subDirPath)
		if err := vm.tree.Append(dir.Path().String(), subDirNode.ID(), subDirNode); err != nil {
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
		return fmt.Errorf("impossible to retrieve the parent directory from path: %s", parentPath)
	}
	childNode := node.NewDirectoryNode(dirToAdd)
	if err := vm.tree.Append(parentNodeItem.(node.DirectoryNode).ID(), childNode.ID(), childNode); err != nil {
		return fmt.Errorf("error appending directory to tree: %w", err)
	}
	return nil
}

func (vm *explorerViewModelImpl) addNewFileToTree(fileToAdd *directory.File) error {
	fileNodePath := fileToAdd.FullPath()
	if _, err := vm.tree.GetValue(fileNodePath); err == nil {
		vm.tree.SetValue(fileNodePath, node.NewFileNode(fileToAdd)) //nolint:errorcheck
		return nil
	}

	newFileNode := node.NewFileNode(fileToAdd)
	if err := vm.tree.Prepend(fileToAdd.DirectoryPath().String(), newFileNode.ID(), newFileNode); err != nil {
		return fmt.Errorf("error appending file to the tree: %w", err)
	}
	return nil
}

func (vm *explorerViewModelImpl) listenEvents() {
	for evt := range vm.bus.Subscribe() {
		switch evt.Type() {
		case connection_deck.SelectEventType.AsSuccess(), connection_deck.UpdateEventType.AsSuccess():
			var conn *connection_deck.Connection
			e, ok := evt.(connection_deck.SelectSuccessEvent)
			if ok {
				conn = e.Connection()
			} else {
				e := evt.(connection_deck.UpdateSuccessEvent)
				conn = e.Connection()
				if conn.ID() != vm.selectedConnectionVal.ID() {
					continue
				}
			}
			hasChanged := (vm.selectedConnectionVal == nil && conn != nil) ||
				(vm.selectedConnectionVal != nil && conn == nil) ||
				(vm.selectedConnectionVal != nil && !vm.selectedConnectionVal.Is(conn))
			if hasChanged {
				vm.selectedConnectionVal = conn
				vm.selectedConnection.Set(conn) //nolint:errcheck
				if err := vm.initializeTreeData(conn); err != nil {
					vm.errorMessage.Set(err.Error()) //nolint:errcheck
					continue
				}
			}

		case connection_deck.RemoveEventType.AsSuccess():
			e := evt.(connection_deck.RemoveSuccessEvent)
			conn := e.Connection()
			if vm.selectedConnectionVal != nil && vm.selectedConnectionVal.Is(conn) {
				vm.selectedConnectionVal = nil
				vm.selectedConnection.Set(nil) //nolint:errcheck
			}

		case directory.ContentUploadedEventType.AsSuccess():
			e := evt.(directory.ContentUploadedSuccessEvent)
			if err := vm.addNewFileToTree(e.File()); err != nil {
				vm.bus.Publish(directory.NewContentUploadedFailureEvent(err, e.Directory()))
				continue
			}
			if err := e.Directory().Notify(e); err != nil {
				vm.notifier.NotifyError(err)
				continue
			}
			fyne.CurrentApp().SendNotification(fyne.NewNotification("File upload", "success"))

		case directory.ContentUploadedEventType.AsFailure():
			e := evt.(directory.ContentUploadedFailureEvent)
			err := fmt.Errorf("error uploading file: %w", e.Error())
			if notifErr := e.Directory().Notify(e); notifErr != nil {
				err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
			}
			vm.notifier.NotifyError(err)
			vm.errorMessage.Set(err.Error()) //nolint:errcheck

		case directory.CreatedEventType.AsSuccess():
			e := evt.(directory.CreatedSuccessEvent)
			if err := vm.addNewDirectoryToTree(e.Directory()); err != nil {
				vm.bus.Publish(directory.NewCreatedFailureEvent(err, e.Parent()))
				continue
			}
			if err := e.Parent().Notify(e); err != nil {
				vm.notifier.NotifyError(err)
			}

		case directory.CreatedEventType.AsFailure():
			e := evt.(directory.CreatedFailureEvent)
			if err := e.Parent().Notify(e); err != nil {
				vm.notifier.NotifyError(err)
				continue
			}
			err := fmt.Errorf("error creating directory: %w", e.Error())
			vm.notifier.NotifyError(err)
			vm.errorMessage.Set(err.Error()) //nolint:errcheck

		case directory.FileDeletedEventType.AsSuccess():
			e := evt.(directory.FileDeletedSuccessEvent)

			if err := vm.tree.Remove(e.File().FullPath()); err != nil {
				vm.bus.Publish(directory.NewFileDeletedFailureEvent(err, e.Parent()))
				continue
			}
			if err := e.Parent().Notify(e); err != nil {
				vm.notifier.NotifyError(err)
				continue
			}
			vm.infoMessage.Set( //nolint:errcheck
				fmt.Sprintf("File %s deleted", e.File().Name()))

		case directory.FileDeletedEventType.AsFailure():
			e := evt.(directory.FileDeletedFailureEvent)
			if err := e.Parent().Notify(e); err != nil {
				vm.notifier.NotifyError(err)
				continue
			}
			err := fmt.Errorf("error deleting file: %w", e.Error())
			vm.notifier.NotifyError(err)
			vm.errorMessage.Set(err.Error()) //nolint:errcheck

		case directory.ContentDownloadEventType.AsSuccess():
			e := evt.(directory.ContentDownloadedSuccessEvent)
			vm.infoMessage.Set( //nolint:errcheck
				fmt.Sprintf("File %s downloaded", e.Content().File().Name()))

		case directory.ContentDownloadEventType.AsFailure():
			e := evt.(directory.ContentDownloadedFailureEvent)
			err := fmt.Errorf("error downloading file: %w", e.Error())
			vm.notifier.NotifyError(err)
			vm.errorMessage.Set(err.Error()) //nolint:errcheck

		case directory.LoadEventType.AsSuccess():
			e := evt.(directory.LoadSuccessEvent)
			if err := vm.handleLoadDirectorySuccess(e); err != nil {
				vm.notifier.NotifyError(err)
				continue
			}

		case directory.LoadEventType.AsFailure():
			e := evt.(directory.LoadFailureEvent)
			dir := e.Directory()
			if err := dir.Notify(e); err != nil {
				vm.notifier.NotifyError(err)
				continue
			}
			dir.SetLoaded(false)
			vm.infoMessage.Set(e.Error().Error()) //nolint:errcheck
		}
	}
}

func (vm *explorerViewModelImpl) handleLoadDirectorySuccess(e directory.LoadSuccessEvent) error {
	dir := e.Directory()
	if err := dir.Notify(e); err != nil {
		return err
	}
	if err := vm.fillSubTree(dir); err != nil {
		dir.SetLoaded(false)
		return fmt.Errorf("error filling sub tree: %w", err)
	}
	return nil
}
