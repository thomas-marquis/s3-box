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

	// LoadDirectory sync a directory with the actual s3 one and load its files and children.
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

	// CreateEmptyFile creates an empty file in the given parent directory
	CreateEmptyFile(parent *directory.Directory, name string)
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
	v := &explorerViewModelImpl{
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

	if err := v.initializeTreeData(initialConnection); err != nil {
		if errors.Is(err, ErrNoConnectionSelected) {
			v.selectedConnection.Set(nil) //nolint:errcheck
			v.selectedConnectionVal = nil
		}
		notifier.NotifyError(fmt.Errorf("error setting initial connection: %w", err))
	}

	bus.SubscribeV2().
		On(
			event.IsOneOf(connection_deck.SelectEventType.AsSuccess(), connection_deck.UpdateEventType.AsSuccess()),
			v.handleConnectionChange).
		On(event.IsOneOf(connection_deck.RemoveEventType.AsSuccess()), v.handleConnectionRemoved).
		On(event.Is(directory.ContentUploadedEventType.AsSuccess()), v.handleFileUploadSuccess).
		On(event.Is(directory.ContentUploadedEventType.AsFailure()), v.handleFileUploadFailure).
		On(event.Is(directory.FileCreatedEventType.AsSuccess()), v.handleCreateFileSuccess).
		On(event.Is(directory.FileCreatedEventType.AsFailure()), v.handleCreateFileFailure).
		On(event.Is(directory.CreatedEventType.AsSuccess()), v.handleCreateDirSuccess).
		On(event.Is(directory.CreatedEventType.AsFailure()), v.handleCreateDirFailure).
		On(event.Is(directory.FileDeletedEventType.AsSuccess()), v.handleDeleteFileSuccess).
		On(event.Is(directory.FileDeletedEventType.AsFailure()), v.handleDeleteFileFailure).
		On(event.Is(directory.ContentDownloadEventType.AsSuccess()), v.handleDownloadFileSuccess).
		On(event.Is(directory.ContentDownloadEventType.AsFailure()), v.handleDownloadFileFailure).
		On(event.Is(directory.LoadEventType.AsSuccess()), v.handleLoadDirSuccess).
		On(event.Is(directory.LoadEventType.AsFailure()), v.handleLoadDirFailure).
		ListenWithWorkers(1)

	return v
}

func (v *explorerViewModelImpl) Tree() binding.Tree[node.Node] {
	return v.tree
}

func (v *explorerViewModelImpl) SelectedConnection() binding.Untyped {
	return v.selectedConnection
}

func (v *explorerViewModelImpl) CurrentSelectedConnection() *connection_deck.Connection {
	return v.selectedConnectionVal
}

func (v *explorerViewModelImpl) LoadDirectory(dirNode node.DirectoryNode) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return err
	}

	if dirNode.Directory().IsLoaded() {
		return nil
	}

	evt, err := dirNode.Directory().Load()
	if err != nil {
		wErr := fmt.Errorf("error loading directory: %w", err)
		v.notifier.NotifyError(wErr)
		return wErr
	}
	v.bus.PublishV2(evt)

	return nil
}

func (v *explorerViewModelImpl) handleLoadDirSuccess(evt event.Event) {
	e := evt.(directory.LoadSuccessEvent)
	dir := e.Directory()
	if err := dir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	if err := v.fillSubTree(dir); err != nil {
		dir.SetLoaded(false)
		v.notifier.NotifyError(fmt.Errorf("error filling sub tree: %w", err))
	}
}

func (v *explorerViewModelImpl) handleLoadDirFailure(evt event.Event) {
	e := evt.(directory.LoadFailureEvent)
	dir := e.Directory()
	if err := dir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	dir.SetLoaded(false)
	v.infoMessage.Set(e.Error().Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) GetFileContent(file *directory.File) (*directory.Content, error) {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return nil, err
	}

	if file.SizeBytes() > v.settingsVm.CurrentFileSizeLimitBytes() {
		err := fmt.Errorf("file is too big to GetFileContent")
		v.notifier.NotifyError(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), v.settingsVm.CurrentTimeout())
	defer cancel()

	content, err := v.directoryRepository.GetFileContent(ctx, v.selectedConnectionVal.ID(), file)
	if err != nil {
		newErr := fmt.Errorf("error getting file content: %w", err)
		v.notifier.NotifyError(newErr)
		return nil, newErr
	}

	return content, nil
}

func (v *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) {
	evt := f.Download(v.selectedConnectionVal.ID(), dest)
	v.bus.PublishV2(evt)
}

func (v *explorerViewModelImpl) handleDownloadFileSuccess(evt event.Event) {
	e := evt.(directory.ContentDownloadedSuccessEvent)
	v.infoMessage.Set( //nolint:errcheck
		fmt.Sprintf("File %s downloaded", e.Content().File().Name()))
}

func (v *explorerViewModelImpl) handleDownloadFileFailure(evt event.Event) {
	e := evt.(directory.ContentDownloadedFailureEvent)
	err := fmt.Errorf("error downloading file: %w", e.Error())
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory, overwrite bool) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		v.bus.PublishV2(directory.NewContentUploadedFailureEvent(err, dir))
		return nil
	}

	evt, err := dir.UploadFile(localPath, overwrite)
	if err != nil {
		if errors.Is(err, directory.ErrAlreadyExists) {
			return err
		}
		err := fmt.Errorf("error uploading file: %w", err)
		v.notifier.NotifyError(err)
		v.bus.PublishV2(directory.NewContentUploadedFailureEvent(err, dir))
		return nil
	}
	v.bus.PublishV2(evt)
	return nil
}

func (v *explorerViewModelImpl) handleFileUploadSuccess(evt event.Event) {
	e := evt.(directory.ContentUploadedSuccessEvent)
	if err := v.addNewFileToTree(e.File()); err != nil {
		v.bus.PublishV2(directory.NewContentUploadedFailureEvent(err, e.Directory()))
		return
	}
	if err := e.Directory().Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	fyne.CurrentApp().SendNotification(fyne.NewNotification("File upload", "success"))
}

func (v *explorerViewModelImpl) handleFileUploadFailure(evt event.Event) {
	e := evt.(directory.ContentUploadedFailureEvent)
	err := fmt.Errorf("error uploading file: %w", e.Error())
	if notifErr := e.Directory().Notify(e); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) DeleteFile(file *directory.File) {
	dirNodeItem, err := v.tree.GetValue(file.DirectoryPath().String())
	if err != nil {
		panic(
			fmt.Sprintf("impossible to retrieve the directory you want to refresh: %s",
				file.DirectoryPath().String()))
	}

	dirNode, ok := dirNodeItem.(node.DirectoryNode)
	if !ok {
		panic(fmt.Sprintf("impossible to cast the item to TreeNode: %s", file.DirectoryPath().String()))
	}

	parent := dirNode.Directory()
	evt, err := parent.RemoveFile(file.Name())
	if err != nil {
		v.bus.PublishV2(directory.NewFileDeletedFailureEvent(
			fmt.Errorf("error removing file from the directory %s: %w", parent.Path(), err), parent))
		return
	}
	v.bus.PublishV2(evt)
}

func (v *explorerViewModelImpl) handleDeleteFileSuccess(evt event.Event) {
	e := evt.(directory.FileDeletedSuccessEvent)

	if err := e.Parent().Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.tree.Remove(e.File().FullPath()); err != nil {
		v.bus.PublishV2(directory.NewFileDeletedFailureEvent(err, e.Parent()))
		return
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File deleted",
		fmt.Sprintf("File %s deleted", e.File().Name())))
}

func (v *explorerViewModelImpl) handleDeleteFileFailure(evt event.Event) {
	e := evt.(directory.FileDeletedFailureEvent)
	if err := e.Parent().Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error deleting file: %w", e.Error())
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) LastDownloadLocation() fyne.ListableURI {
	return v.lastDownloadLocation
}

func (v *explorerViewModelImpl) UpdateLastDownloadLocation(filePath string) error {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		wErr := fmt.Errorf("update download location: %w", err)
		v.notifier.NotifyError(wErr)
		return wErr
	}
	v.lastDownloadLocation = uriLister
	return nil
}

func (v *explorerViewModelImpl) LastUploadLocation() fyne.ListableURI {
	return v.lastUploadDir
}

func (v *explorerViewModelImpl) UpdateLastUploadLocation(filePath string) {
	dirPath := filepath.Dir(filePath)
	uri := storage.NewFileURI(dirPath)
	uriLister, err := storage.ListerForURI(uri)
	if err != nil {
		v.notifier.NotifyError(fmt.Errorf("update upload location: %w", err))
		return
	}
	v.lastUploadDir = uriLister
}

func (v *explorerViewModelImpl) CreateEmptyDirectory(parent *directory.Directory, name string) {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		v.bus.PublishV2(directory.NewCreatedFailureEvent(err, parent))
		return
	}

	evt, err := parent.NewSubDirectory(name)
	if err != nil {
		wErr := fmt.Errorf("error creating subdirectory: %w", err)
		v.notifier.NotifyError(wErr)
		v.bus.PublishV2(directory.NewCreatedFailureEvent(wErr, parent))
		return
	}

	v.bus.PublishV2(evt)
}

func (v *explorerViewModelImpl) handleCreateDirSuccess(evt event.Event) {
	e := evt.(directory.CreatedSuccessEvent)
	if err := v.addNewDirectoryToTree(e.Directory()); err != nil {
		v.bus.PublishV2(directory.NewCreatedFailureEvent(err, e.Parent()))
		return
	}
	if err := e.Parent().Notify(e); err != nil {
		v.notifier.NotifyError(err)
	}
}

func (v *explorerViewModelImpl) handleCreateDirFailure(evt event.Event) {
	e := evt.(directory.CreatedFailureEvent)
	if err := e.Parent().Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error creating directory: %w", e.Error())
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) CreateEmptyFile(parent *directory.Directory, name string) {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		v.bus.PublishV2(directory.NewCreatedFailureEvent(err, parent))
		return
	}

	evt, err := parent.NewFile(name, false)
	if err != nil {
		wErr := fmt.Errorf("error creating file: %w", err)
		v.notifier.NotifyError(wErr)
		v.bus.PublishV2(directory.NewCreatedFailureEvent(wErr, parent))
		return
	}

	v.bus.PublishV2(evt)
}

func (v *explorerViewModelImpl) handleCreateFileSuccess(evt event.Event) {
	e := evt.(directory.FileCreatedSuccessEvent)
	if err := v.addNewFileToTree(e.File()); err != nil {
		v.bus.PublishV2(directory.NewFileCreatedFailureEvent(err, e.Directory()))
		return
	}
	if err := e.Directory().Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
}

func (v *explorerViewModelImpl) handleCreateFileFailure(evt event.Event) {
	e := evt.(directory.FileCreatedFailureEvent)
	err := fmt.Errorf("error creating file: %w", e.Error())
	if notifErr := e.Directory().Notify(e); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) initializeTreeData(c *connection_deck.Connection) error {
	v.tree = binding.NewTree[node.Node](func(n1 node.Node, n2 node.Node) bool {
		return n1.ID() == n2.ID()
	})

	if c == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return err
	}

	displayLabel := "Bucket: " + c.Bucket()

	rootDir, err := directory.New(c.ID(), directory.RootDirName, directory.NilParentPath)
	if err != nil {
		newErr := fmt.Errorf("error initializing the root directory: %w", err)
		v.notifier.NotifyError(newErr)
		return newErr
	}
	rootNode := node.NewDirectoryNode(rootDir, node.WithDisplayName(displayLabel))
	if err := v.tree.Append("", rootNode.ID(), rootNode); err != nil {
		newErr := fmt.Errorf("error appending directory to tree: %w", err)
		v.notifier.NotifyError(newErr)
		return newErr
	}

	if err := v.LoadDirectory(rootNode); err != nil {
		newErr := fmt.Errorf("error loading root directory: %w", err)
		v.notifier.NotifyError(newErr)
		return newErr
	}

	return nil
}

func (v *explorerViewModelImpl) fillSubTree(dir *directory.Directory) error {
	files, err := dir.Files()
	if err != nil {
		v.notifier.NotifyError(fmt.Errorf("error getting files: %w", err))
		return err
	}

	subDirs, err := dir.SubDirectories()
	if err != nil {
		v.notifier.NotifyError(fmt.Errorf("error getting subdirectories: %w", err))
		return err
	}

	for _, file := range files {
		fileNode := node.NewFileNode(file)
		if err := v.tree.Append(dir.Path().String(), fileNode.ID(), fileNode); err != nil {
			v.notifier.NotifyError(fmt.Errorf("error appending file to tree: %w", err))
			continue
		}
	}

	for _, subDirPath := range subDirs {
		subDirNode := node.NewDirectoryNode(subDirPath)
		if err := v.tree.Append(dir.Path().String(), subDirNode.ID(), subDirNode); err != nil {
			v.notifier.NotifyError(fmt.Errorf("error appending subdirectory to tree: %w", err))
			continue
		}
	}

	return nil
}

func (v *explorerViewModelImpl) addNewDirectoryToTree(dirToAdd *directory.Directory) error {
	parentPath := dirToAdd.Path().ParentPath()
	parentNodeItem, err := v.tree.GetValue(parentPath.String())
	if err != nil {
		return fmt.Errorf("impossible to retrieve the parent directory from path: %s", parentPath)
	}
	childNode := node.NewDirectoryNode(dirToAdd)
	if err := v.tree.Append(parentNodeItem.(node.DirectoryNode).ID(), childNode.ID(), childNode); err != nil {
		return fmt.Errorf("error appending directory to tree: %w", err)
	}
	return nil
}

func (v *explorerViewModelImpl) addNewFileToTree(fileToAdd *directory.File) error {
	fileNodePath := fileToAdd.FullPath()
	if _, err := v.tree.GetValue(fileNodePath); err == nil {
		v.tree.SetValue(fileNodePath, node.NewFileNode(fileToAdd)) //nolint:errcheck
		return nil
	}

	newFileNode := node.NewFileNode(fileToAdd)
	if err := v.tree.Prepend(fileToAdd.DirectoryPath().String(), newFileNode.ID(), newFileNode); err != nil {
		return fmt.Errorf("error appending file to the tree: %w", err)
	}
	return nil
}

func (v *explorerViewModelImpl) handleConnectionChange(evt event.Event) {
	var conn *connection_deck.Connection
	e, ok := evt.(connection_deck.SelectSuccessEvent)
	if ok {
		conn = e.Connection()
	} else {
		e := evt.(connection_deck.UpdateSuccessEvent)
		conn = e.Connection()
		if conn.ID() != v.selectedConnectionVal.ID() {
			return
		}
	}
	hasChanged := (v.selectedConnectionVal == nil && conn != nil) ||
		(v.selectedConnectionVal != nil && conn == nil) ||
		(v.selectedConnectionVal != nil && !v.selectedConnectionVal.Is(conn))
	if hasChanged {
		v.selectedConnectionVal = conn
		v.selectedConnection.Set(conn) //nolint:errcheck
		if err := v.initializeTreeData(conn); err != nil {
			v.errorMessage.Set(err.Error()) //nolint:errcheck
			return
		}
	}
}

func (v *explorerViewModelImpl) handleConnectionRemoved(evt event.Event) {
	e := evt.(connection_deck.RemoveSuccessEvent)
	conn := e.Connection()
	if v.selectedConnectionVal != nil && v.selectedConnectionVal.Is(conn) {
		v.selectedConnectionVal = nil
		v.selectedConnection.Set(nil) //nolint:errcheck
	}
}
