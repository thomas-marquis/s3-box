package viewmodel

import (
	"errors"
	"sync"

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

const (
	maxPendingUserValidations = 30
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

	SelectedDirectory() *directory.Directory
	SetSelectedDirectory(dir *directory.Directory)
	IsSelectedDirectoryLoading() binding.Bool

	PendingUserValidations() <-chan directory.UserValidationEvent

	// AddStateListener registers a callback function to be notified of any changes in directories or files.
	AddStateListener(func())

	////////////////////////
	// Action methods
	////////////////////////

	// LoadDirectory sync a directory with the actual s3 one and load its files and children.
	// If the directory is already open, it will do nothing.
	LoadDirectory(dirNode node.DirectoryNode) error

	ReloadDirectory(dir *directory.Directory) error

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

	// RenameDirectory renames a directory
	RenameDirectory(dir *directory.Directory, newName string)

	// RenameFile renames a file
	RenameFile(file *directory.File, newName string)

	Validate(event directory.UserValidationEvent, validated bool)

	ResumeRename(dir *directory.Directory) error
	RollbackRename(dir *directory.Directory) error
	AbortRename(dir *directory.Directory) error
}

type explorerViewModelImpl struct {
	baseViewModel
	sync.Mutex

	tree binding.Tree[node.Node]

	selectedConnection    binding.Untyped
	selectedConnectionVal *connection_deck.Connection

	settingsVm           SettingsViewModel
	lastDownloadLocation fyne.ListableURI
	lastUploadDir        fyne.ListableURI

	selectedDirectory    *directory.Directory
	isSelectedDirLoading binding.Bool

	pendingUserValidations chan directory.UserValidationEvent

	stateListeners []func()

	notifier notification.Repository
	bus      event.Bus
}

func NewExplorerViewModel(
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
		settingsVm:             settingsVm,
		notifier:               notifier,
		selectedConnectionVal:  initialConnection,
		selectedConnection:     binding.NewUntyped(),
		bus:                    bus,
		selectedDirectory:      nil,
		isSelectedDirLoading:   binding.NewBool(),
		pendingUserValidations: make(chan directory.UserValidationEvent, maxPendingUserValidations),
		stateListeners:         make([]func(), 0),
	}

	if err := v.initializeTreeData(initialConnection); err != nil {
		if errors.Is(err, ErrNoConnectionSelected) {
			v.selectedConnection.Set(nil) //nolint:errcheck
			v.selectedConnectionVal = nil
		}
		notifier.NotifyError(fmt.Errorf("error setting initial connection: %w", err))
	}

	bus.Subscribe().
		On(event.IsOneOf(
			connection_deck.SelectEventType.AsSuccess(),
			connection_deck.UpdateEventType.AsSuccess(),
		), v.handleConnectionChange).
		On(event.IsOneOf(connection_deck.RemoveEventType.AsSuccess()), v.handleConnectionRemoved).
		On(event.Is(directory.FileUploadEventType.AsSuccess()), v.handleFileUploadSuccess).
		On(event.Is(directory.FileUploadEventType.AsFailure()), v.handleFileUploadFailure).
		On(event.Is(directory.CreateFileSucceededEventType), v.handleCreateFileSuccess).
		On(event.Is(directory.CreateFileFailedEventType), v.handleCreateFileFailure).
		On(event.Is(directory.CreateEventType.AsSuccess()), v.handleCreateDirSuccess).
		On(event.Is(directory.CreateEventType.AsFailure()), v.handleCreateDirFailure).
		On(event.Is(directory.DeleteFileSucceededEventType), v.handleDeleteFileSuccess).
		On(event.Is(directory.DeleteFileFailedEventType), v.handleDeleteFileFailure).
		On(event.Is(directory.FileDownloadEventType.AsSuccess()), v.handleDownloadFileSuccess).
		On(event.Is(directory.FileDownloadEventType.AsFailure()), v.handleDownloadFileFailure).
		On(event.Is(directory.LoadEventType.AsSuccess()), v.handleLoadDirSuccess).
		On(event.Is(directory.LoadEventType.AsFailure()), v.handleLoadDirFailure).
		On(event.Is(directory.RenameEventType.AsSuccess()), v.handleRenameDirectorySuccess).
		On(event.Is(directory.RenameEventType.AsFailure()), v.handleRenameDirectoryFailure).
		On(event.Is(directory.FileRenameEventType.AsSuccess()), v.handleRenameFileSuccess).
		On(event.Is(directory.FileRenameEventType.AsFailure()), v.handleRenameFileFailure).
		On(event.Is(directory.UserValidationEventType), v.handleUserValidationRequest).
		On(event.Is(directory.UserValidationRefusedEventType), v.handleUserValidationRefused).
		ListenWithWorkers(1)

	return v
}

func (v *explorerViewModelImpl) AddStateListener(listener func()) {
	v.stateListeners = append(v.stateListeners, listener)
}

func (v *explorerViewModelImpl) triggerStateListeners() {
	fyne.Do(func() {
		for _, listener := range v.stateListeners {
			listener()
		}
	})
}

func (v *explorerViewModelImpl) Validate(event directory.UserValidationEvent, accepted bool) {
	if accepted {
		v.bus.Publish(event.NewAcceptedEvent())
	} else {
		v.bus.Publish(event.NewRefusedEvent())
	}
}

func (v *explorerViewModelImpl) PendingUserValidations() <-chan directory.UserValidationEvent {
	return v.pendingUserValidations
}

func (v *explorerViewModelImpl) handleUserValidationRequest(evt event.Event) {
	e := evt.(directory.UserValidationEvent)
	v.pendingUserValidations <- e
}

func (v *explorerViewModelImpl) handleUserValidationRefused(evt event.Event) {
	e := evt.(directory.UserValidationRefusedEvent)
	dir := e.Directory

	if err := dir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if dir.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) Tree() binding.Tree[node.Node] {
	return v.tree
}

func (v *explorerViewModelImpl) SelectedConnection() binding.Untyped {
	return v.selectedConnection
}

func (v *explorerViewModelImpl) CurrentSelectedConnection() *connection_deck.Connection {
	v.Lock()
	defer v.Unlock()
	return v.selectedConnectionVal
}

func (v *explorerViewModelImpl) SelectedDirectory() *directory.Directory {
	return v.selectedDirectory
}

func (v *explorerViewModelImpl) SetSelectedDirectory(dir *directory.Directory) {
	v.selectedDirectory = dir
}

func (v *explorerViewModelImpl) IsSelectedDirectoryLoading() binding.Bool {
	return v.isSelectedDirLoading
}

func (v *explorerViewModelImpl) LoadDirectory(dirNode node.DirectoryNode) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return err
	}

	evt, err := dirNode.Directory().Load()
	if err != nil {
		wErr := fmt.Errorf("impossible to (re)load the directory: %w", err)
		v.notifier.NotifyError(wErr)
		return wErr
	}
	v.isSelectedDirLoading.Set(true) // nolint:errcheck
	v.bus.Publish(evt)

	return nil
}

func (v *explorerViewModelImpl) ReloadDirectory(dir *directory.Directory) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return err
	}

	var subNodePaths []string
	for _, sd := range dir.SubDirectories() {
		subNodePaths = append(subNodePaths, sd.Path().String())
	}
	for _, f := range dir.Files() {
		subNodePaths = append(subNodePaths, f.FullPath())
	}

	evt, err := dir.Load()
	if err != nil {
		wErr := fmt.Errorf("impossible to (re)load the directory: %w", err)
		v.notifier.NotifyError(wErr)
		return wErr
	}
	v.bus.Publish(evt)

	v.isSelectedDirLoading.Set(true) // nolint:errcheck

	for _, p := range subNodePaths {
		if err := v.tree.Remove(p); err != nil {
			v.notifier.NotifyError(fmt.Errorf("error removing directory node: %w", err))
			return nil
		}
	}

	return nil
}

func (v *explorerViewModelImpl) handleLoadDirSuccess(evt event.Event) {
	e := evt.(directory.LoadSuccessEvent)
	dir := e.Directory
	if err := dir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.fillSubTree(dir); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error filling sub tree: %w", err))
	}

	if dir.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleLoadDirFailure(evt event.Event) {
	e := evt.(directory.LoadFailureEvent)
	dir := e.Directory
	if err := dir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.infoMessage.Set(e.Error().Error()) //nolint:errcheck

	if dir.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) {
	evt := f.Download(v.selectedConnectionVal.ID(), dest)
	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleDownloadFileSuccess(evt event.Event) {
	e := evt.(directory.FileDownloadSuccessEvent)
	v.infoMessage.Set( //nolint:errcheck
		fmt.Sprintf("File %s downloaded", e.File.Name()))
}

func (v *explorerViewModelImpl) handleDownloadFileFailure(evt event.Event) {
	e := evt.(directory.FileDownloadFailureEvent)
	err := fmt.Errorf("error downloading file: %w", e.Error())
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) UploadFile(localPath string, dir *directory.Directory, overwrite bool) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		v.bus.Publish(directory.NewFileUploadFailureEvent(err, dir))
		return nil
	}

	evt, err := dir.UploadFile(localPath, overwrite)
	if err != nil {
		if errors.Is(err, directory.ErrAlreadyExists) {
			return err
		}
		err := fmt.Errorf("error uploading file: %w", err)
		v.notifier.NotifyError(err)
		v.bus.Publish(directory.NewFileUploadFailureEvent(err, dir))
		return nil
	}
	v.bus.Publish(evt)
	return nil
}

func (v *explorerViewModelImpl) handleFileUploadSuccess(evt event.Event) {
	e := evt.(directory.FileUploadSuccessEvent)
	if err := v.addNewFileToTree(e.File); err != nil {
		v.bus.Publish(directory.NewFileUploadFailureEvent(err, e.Directory))
		return
	}
	if err := e.Directory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	fyne.CurrentApp().SendNotification(fyne.NewNotification("File upload", "success"))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleFileUploadFailure(evt event.Event) {
	e := evt.(directory.FileUploadFailureEvent)
	err := fmt.Errorf("error uploading file: %w", e.Error())
	if notifErr := e.Directory.Notify(e); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
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
		return
	}
	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleDeleteFileSuccess(evt event.Event) {
	e := evt.(directory.DeleteFileSucceeded)

	if err := e.ParentDirectory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.tree.Remove(e.File.FullPath()); err != nil {
		v.bus.Publish(e.NewFailureEvent(err))
		return
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File deleted",
		fmt.Sprintf("File %s deleted", e.File.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleDeleteFileFailure(evt event.Event) {
	e := evt.(directory.DeleteFileFailed)
	if err := e.ParentDirectory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error deleting file: %w", e.Error())
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
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
		return
	}

	evt, err := parent.NewSubDirectory(name)
	if err != nil {
		wErr := fmt.Errorf("error creating subdirectory: %w", err)
		v.notifier.NotifyError(wErr)
		return
	}

	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleCreateDirSuccess(evt event.Event) {
	e := evt.(directory.CreateSuccessEvent)
	if err := v.addNewDirectoryToTree(e.Directory); err != nil {
		v.bus.Publish(e.NewFailureEvent(err))
		return
	}
	if err := e.ParentDirectory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
	}
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleCreateDirFailure(evt event.Event) {
	e := evt.(directory.CreatedFailureEvent)
	if err := e.ParentDirectory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error creating directory: %w", e.Error())
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) CreateEmptyFile(parent *directory.Directory, name string) {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return
	}

	evt, err := parent.NewFile(name, false)
	if err != nil {
		wErr := fmt.Errorf("error creating file: %w", err)
		v.notifier.NotifyError(wErr)
		return
	}

	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleCreateFileSuccess(evt event.Event) {
	e := evt.(directory.CreateFileSucceeded)
	if err := v.addNewFileToTree(e.File); err != nil {
		v.bus.Publish(e.NewFailureEvent(err))
		return
	}
	if err := e.Directory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleCreateFileFailure(evt event.Event) {
	e := evt.(directory.CreateFileFailed)
	err := fmt.Errorf("error creating file: %w", e.Error())
	if notifErr := e.Directory.Notify(e); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) RenameDirectory(dir *directory.Directory, newName string) {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return
	}

	evt, err := dir.Rename(newName)
	if err != nil {
		wErr := fmt.Errorf("error renaming directory: %w", err)
		v.notifier.NotifyError(wErr)
		return
	}

	v.isSelectedDirLoading.Set(true) // nolint:errcheck
	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleRenameDirectorySuccess(evt event.Event) {
	e := evt.(directory.RenameSuccessEvent)
	dir := e.Directory

	defer func() {
		if dir.Is(v.selectedDirectory) {
			v.isSelectedDirLoading.Set(false) // nolint:errcheck
		}
	}()

	oldPath := dir.Path().String()

	if err := dir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.tree.Remove(oldPath); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error removing old directory node: %w", err))
		return
	}

	var (
		n   node.Node
		err error
	)
	_, err = v.tree.GetValue(dir.Path().String())
	n = node.NewDirectoryNode(dir)
	if err != nil {
		if err := v.tree.Prepend(dir.ParentPath().String(), n.ID(), n); err != nil {
			v.notifier.NotifyError(fmt.Errorf("error adding new directory node: %w", err))
			return
		}
	} else {
		v.tree.SetValue(dir.Path().String(), n) //nolint:errcheck
	}
	newDirNode := n.(node.DirectoryNode)

	if err := v.LoadDirectory(newDirNode); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error loading the renamed directory: %w", err))
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("Directory renamed",
		fmt.Sprintf("Directory %s renamed to %s", oldPath, dir.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleRenameDirectoryFailure(evt event.Event) {
	e := evt.(directory.RenameFailureEvent)
	dir := e.Directory

	defer func() {
		if dir.Is(v.selectedDirectory) {
			v.isSelectedDirLoading.Set(false) // nolint:errcheck
		}
	}()

	err := fmt.Errorf("error renaming directory: %w", e.Error())
	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) ResumeRename(dir *directory.Directory) error {
	evt, err := dir.Recover(directory.RecoveryChoiceRenameResume)
	if err != nil {
		return fmt.Errorf("impossible to resume rename: %w", err)
	}
	v.bus.Publish(evt)
	v.isSelectedDirLoading.Set(true) // nolint:errcheck
	return nil
}

func (v *explorerViewModelImpl) RollbackRename(dir *directory.Directory) error {
	evt, err := dir.Recover(directory.RecoveryChoiceRenameRollback)
	if err != nil {
		return fmt.Errorf("impossible to rollback rename: %w", err)
	}
	v.bus.Publish(evt)
	v.isSelectedDirLoading.Set(true) // nolint:errcheck
	return nil
}

func (v *explorerViewModelImpl) AbortRename(dir *directory.Directory) error {
	evt, err := dir.Recover(directory.RecoveryChoiceRenameAbort)
	if err != nil {
		return fmt.Errorf("impossible to abort rename: %w", err)
	}
	v.bus.Publish(evt)
	v.isSelectedDirLoading.Set(true) // nolint:errcheck
	return nil
}

func (v *explorerViewModelImpl) RenameFile(file *directory.File, newName string) {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		v.bus.Publish(directory.NewFileRenameFailureEvent(err, nil, file, newName))
		return
	}

	evt, err := file.Rename(newName)
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}

	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleRenameFileSuccess(evt event.Event) {
	e := evt.(directory.FileRenameSuccessEvent)
	file := e.File
	parentDir := e.Directory

	oldFullPath := file.FullPath()

	// Update the parent directory's state
	if err := parentDir.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	// Remove the old file node from the tree
	if err := v.tree.Remove(oldFullPath); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error removing old file node: %w", err))
		return
	}

	// Add the new file node to the tree
	newFileNode := node.NewFileNode(file)
	if err := v.tree.Append(file.DirectoryPath().String(), newFileNode.ID(), newFileNode); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error adding new file node: %w", err))
		return
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File renamed",
		fmt.Sprintf("File renamed to %s", file.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleRenameFileFailure(evt event.Event) {
	e := evt.(directory.FileRenameFailureEvent)
	err := fmt.Errorf("error renaming file: %w", e.Error())
	if err := e.Directory.Notify(e); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
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

	rootDir, err := directory.NewRoot(c.ID())
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
	files := dir.Files()
	subDirs := dir.SubDirectories()

	for _, subDir := range subDirs {
		subDirNode := node.NewDirectoryNode(subDir)
		if err := v.tree.Append(dir.Path().String(), subDirNode.ID(), subDirNode); err != nil {
			v.notifier.NotifyError(fmt.Errorf("error appending subdirectory to tree: %w", err))
			continue
		}
		if err := v.fillSubTree(subDir); err != nil {
			return err
		}
	}

	for _, file := range files {
		fileNode := node.NewFileNode(file)
		if err := v.tree.Append(dir.Path().String(), fileNode.ID(), fileNode); err != nil {
			v.notifier.NotifyError(fmt.Errorf("error appending file to tree: %w", err))
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
	if err := v.tree.Prepend(parentNodeItem.(node.DirectoryNode).ID(), childNode.ID(), childNode); err != nil {
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
	if err := v.tree.Append(fileToAdd.DirectoryPath().String(), newFileNode.ID(), newFileNode); err != nil {
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
		v.Lock()
		v.selectedConnectionVal = conn
		v.selectedConnection.Set(conn) //nolint:errcheck
		v.Unlock()

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
