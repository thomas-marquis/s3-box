package viewmodel

import (
	"errors"
	"io/fs"
	"os"
	"sync"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"

	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

const (
	maxPendingUserValidations = 30
)

type UploadPreviewState struct {
	Preview *directory.Preview
	BaseUri string
}

// ExplorerViewModel represents the view model for the file explorer interface.
// It handles the tree structure display, file operations, and directory management
// while maintaining the connection with the underlying storage system.
type ExplorerViewModel interface {
	ViewModel

	////////////////////////
	// State methods
	////////////////////////

	SelectedConnection() binding.Untyped

	CurrentSelectedConnection() *connection_deck.Connection

	// LastDownloadLocation returns the URI of the last used save directory
	LastDownloadLocation() fyne.ListableURI

	// LastUploadLocation returns the URI of the last used upload directory
	LastUploadLocation() fyne.ListableURI

	SelectedDirectory() *directory.Directory
	SetSelectedDirectory(dir *directory.Directory)
	IsSelectedDirectoryLoading() binding.Bool

	PendingUserValidations() <-chan directory.UserValidationAsked

	// AddStateListener registers a callback function to be notified of any changes in directories or files.
	AddStateListener(func())

	// OnUploadReady registers a callback function to be notified when the upload is ready.
	OnUploadReady(func(previewState UploadPreviewState))

	////////////////////////
	// Action methods
	////////////////////////

	// LoadDirectory sync a directory with the actual s3 one and load its files and children.
	// If the directory is already open, it will do nothing.
	LoadDirectory(dir *directory.Directory) error

	ReloadDirectory(dir *directory.Directory) error

	// DownloadFile downloads a file to the specified local destination
	DownloadFile(f *directory.File, dest string)

	PrepareUpload(uris []fyne.URI, dir *directory.Directory) error
	DoUpload(localBasePath string, preview *directory.Preview, strategy directory.MaterializeStrategy)
	UploadOne(localPath string, dir *directory.Directory, overwrite bool) error

	// DeleteFile removes a file from storage and updates the tree
	DeleteFile(file *directory.File)

	DeleteDirectory(dir *directory.Directory)

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

	Validate(event directory.UserValidationAsked, validated bool)

	ResumeRename(dir *directory.Directory) error
	RollbackRename(dir *directory.Directory) error
	AbortRename(dir *directory.Directory) error
}

type explorerViewModelImpl struct {
	baseViewModel
	sync.Mutex

	selectedConnection    binding.Untyped
	selectedConnectionVal *connection_deck.Connection

	settingsVm           SettingsViewModel
	lastDownloadLocation fyne.ListableURI
	lastUploadDir        fyne.ListableURI

	selectedDirectory    *directory.Directory
	isSelectedDirLoading binding.Bool

	pendingUserValidations chan directory.UserValidationAsked

	stateListeners []func()
	onUploadReady  func(previewState UploadPreviewState)

	notifier notification.Repository
	bus      event.Bus

	state *state.State
}

func NewExplorerViewModel(
	settingsVm SettingsViewModel,
	notifier notification.Repository,
	initialConnection *connection_deck.Connection,
	bus event.Bus,
	st *state.State,
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
		pendingUserValidations: make(chan directory.UserValidationAsked, maxPendingUserValidations),
		stateListeners:         make([]func(), 0),
		state:                  st,
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
			connection_deck.SelectConnectionSucceededType,
			connection_deck.UpdateConnectionSucceededType,
		), v.handleConnectionChange).
		On(event.Is(connection_deck.RemoveConnectionSucceededType), v.handleConnectionRemoved).
		On(event.Is(directory.UploadFileSucceededType), v.handleFileUploadSuccess).
		On(event.Is(directory.UploadFileFailedType), v.handleFileUploadFailure).
		On(event.Is(directory.CreateFileSucceededType), v.handleCreateFileSuccess).
		On(event.Is(directory.CreateFileFailedType), v.handleCreateFileFailure).
		On(event.Is(directory.CreateSucceededType), v.handleCreateDirSuccess).
		On(event.Is(directory.CreateFailedType), v.handleCreateDirFailure).
		On(event.Is(directory.DeleteFileSucceededType), v.handleDeleteFileSuccess).
		On(event.Is(directory.DeleteFileFailedType), v.handleDeleteFileFailure).
		On(event.Is(directory.DownloadFileSucceededType), v.handleDownloadFileSuccess).
		On(event.Is(directory.DownloadFileFailedType), v.handleDownloadFileFailure).
		On(event.Is(directory.LoadSucceededType), v.handleLoadDirSuccess).
		On(event.Is(directory.LoadFailedType), v.handleLoadDirFailure).
		On(event.Is(directory.RenameSucceededType), v.handleRenameDirectorySuccess).
		On(event.Is(directory.RenameFailedType), v.handleRenameDirectoryFailure).
		On(event.Is(directory.RenameFileSucceededType), v.handleRenameFileSuccess).
		On(event.Is(directory.RenameFileFailedType), v.handleRenameFileFailure).
		On(event.Is(directory.UserValidationAskedType), v.handleUserValidationRequest).
		On(event.Is(directory.UserValidationRefusedType), v.handleUserValidationRefused).
		On(event.Is(directory.UploadReadyType), v.handleUploadReady).
		On(event.Is(directory.DeleteFailedType), v.handleDeleteDirectoryFailure).
		On(event.Is(directory.DeleteSucceededType), v.handleDeleteDirectorySuccess).
		ListenWithWorkers(3)

	return v
}

func (v *explorerViewModelImpl) AddStateListener(listener func()) {
	v.stateListeners = append(v.stateListeners, listener)
}

func (v *explorerViewModelImpl) OnUploadReady(listener func(previewState UploadPreviewState)) {
	v.onUploadReady = listener
}

func (v *explorerViewModelImpl) triggerStateListeners() {
	fyne.Do(func() {
		for _, listener := range v.stateListeners {
			listener()
		}
	})
}

func (v *explorerViewModelImpl) Validate(evt directory.UserValidationAsked, accepted bool) {
	if accepted {
		v.bus.Publish(event.New(directory.UserValidationAccepted{
			Directory: evt.Directory,
			Reason:    evt.Reason,
		}))
	} else {
		v.bus.Publish(event.New(directory.UserValidationRefused{
			Directory: evt.Directory,
			Reason:    evt.Reason,
		}))
	}
}

func (v *explorerViewModelImpl) PendingUserValidations() <-chan directory.UserValidationAsked {
	return v.pendingUserValidations
}

func (v *explorerViewModelImpl) handleUserValidationRequest(evt event.Event) {
	pl := evt.Payload().(directory.UserValidationAsked)
	v.pendingUserValidations <- pl
}

func (v *explorerViewModelImpl) handleUserValidationRefused(evt event.Event) {
	pl := evt.Payload().(directory.UserValidationRefused)
	dir := pl.Directory

	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if dir.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	v.triggerStateListeners()
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

func (v *explorerViewModelImpl) LoadDirectory(dir *directory.Directory) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return err
	}

	evt, err := dir.Load()
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

	evt, err := dir.Load()
	if err != nil {
		wErr := fmt.Errorf("impossible to (re)load the directory: %w", err)
		v.notifier.NotifyError(wErr)
		return wErr
	}
	v.bus.Publish(evt)

	v.isSelectedDirLoading.Set(true) // nolint:errcheck

	return nil
}

func (v *explorerViewModelImpl) handleLoadDirSuccess(evt event.Event) {
	pl := evt.Payload().(directory.LoadSucceeded)
	dir := pl.Directory
	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	v.state.Explorer().UpdateChildren(dir)

	if dir.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleLoadDirFailure(evt event.Event) {
	pl := evt.Payload().(directory.LoadFailed)
	dir := pl.Directory
	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.infoMessage.Set(pl.Err.Error()) //nolint:errcheck

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
	pl := evt.Payload().(directory.DownloadFileSucceeded)
	v.infoMessage.Set( //nolint:errcheck
		fmt.Sprintf("File %s downloaded", pl.File.Name()))
}

func (v *explorerViewModelImpl) handleDownloadFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.DownloadFileFailed)
	err := fmt.Errorf("error downloading file: %w", pl.Err)
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
}

func (v *explorerViewModelImpl) DoUpload(localBasePath string, preview *directory.Preview, strategy directory.MaterializeStrategy) {
	uploadMat := directory.NewUploadMaterializer(preview, localBasePath)
	v.bus.Publish(uploadMat.Materialize(strategy))
}

func (v *explorerViewModelImpl) UploadOne(localPath string, dir *directory.Directory, overwrite bool) error {
	if v.selectedConnectionVal == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return nil
	}

	evt, err := dir.UploadFile(localPath, overwrite)
	if err != nil {
		if errors.Is(err, directory.ErrAlreadyExists) {
			return err
		}
		err := fmt.Errorf("error uploading file: %w", err)
		v.notifier.NotifyError(err)
		return nil
	}
	v.bus.Publish(evt)
	return nil
}

func (v *explorerViewModelImpl) handleFileUploadSuccess(evt event.Event) {
	pl := evt.Payload().(directory.UploadFileSucceeded)
	if err := v.state.Explorer().UpdateOrAppendFile(pl.File); err != nil {
		v.bus.Publish(evt.NewFollowup(directory.UploadFileFailed{
			Err:       err,
			Directory: pl.Directory,
		}))
		return
	}
	if err := pl.Directory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	fyne.CurrentApp().SendNotification(fyne.NewNotification("File upload", "success"))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleFileUploadFailure(evt event.Event) {
	pl := evt.Payload().(directory.UploadFileFailed)
	err := fmt.Errorf("error uploading file: %w", pl.Err)
	if notifErr := pl.Directory.Notify(evt); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) PrepareUpload(uris []fyne.URI, dir *directory.Directory) error {
	prev, err := makePreviewFromUris(uiutils.FromFyneUrisToPaths(uris), dir)
	if err != nil {
		return err
	}

	loadMat := directory.NewLoadMaterializer(prev, directory.UploadReady{
		Directory: dir,
		SrcPaths:  uiutils.FromFyneUrisToPaths(uris),
	}, directory.UploadFailed{
		Err:       errors.New("timeout"),
		Directory: dir,
	})
	v.bus.Publish(loadMat.Materialize(directory.MaterializeReplace))

	return nil
}

func makePreviewFromUris(paths []string, dir *directory.Directory) (*directory.Preview, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths provided")
	}

	prev, err := dir.Preview()
	if err != nil {
		return nil, err
	}
	prevsByPath := make(map[string]*directory.Preview)

	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if fi.IsDir() {
			dirPrev, err := prev.AddSubDirectory(fi.Name())
			if err != nil {
				return nil, err
			}
			prevsByPath[p] = dirPrev

			if err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if path == p {
					return nil
				}
				parentPath := filepath.Dir(path)

				parentPrev := prevsByPath[parentPath]
				if d.IsDir() {
					subprev, err := parentPrev.AddSubDirectory(d.Name())
					if err != nil {
						return err
					}
					prevsByPath[path] = subprev
				} else {
					fii, err := d.Info()
					if err != nil {
						return err
					}
					if err := parentPrev.AddFile(d.Name(), uint64(fii.Size()), fii.ModTime()); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				return nil, err
			}
		} else {
			if err := prev.AddFile(fi.Name(), uint64(fi.Size()), fi.ModTime()); err != nil {
				return nil, err
			}
		}
	}
	return prev, nil
}

func (v *explorerViewModelImpl) handleUploadReady(evt event.Event) {
	pl := evt.Payload().(directory.UploadReady)

	localParentDirUri := uiutils.GetCommonParentPath(pl.SrcPaths)

	prev, err := makePreviewFromUris(pl.SrcPaths, pl.Directory)
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}
	if v.onUploadReady != nil {
		v.onUploadReady(UploadPreviewState{ //nolint:errcheck
			Preview: prev,
			BaseUri: localParentDirUri,
		})
	}
}

func (v *explorerViewModelImpl) DeleteDirectory(dir *directory.Directory) {
	if directory.RootPath.Is(dir) {
		v.errorMessage.Set("Cannot delete root directory") //nolint:errcheck
		return
	}

	parent := dir.Parent()

	evt, err := parent.RemoveSubDirectory(dir.Name())
	if err != nil {
		v.errorMessage.Set(err.Error()) //nolint:errcheck
		return
	}

	v.isSelectedDirLoading.Set(true) // nolint:errcheck

	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleDeleteDirectorySuccess(evt event.Event) {
	pl := evt.Payload().(directory.DeleteSucceeded)

	if err := pl.Parent.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.state.Explorer().RemoveNode(pl.Directory.Path().String()); err != nil {
		v.bus.Publish(evt.NewFollowup(directory.DeleteFailed{
			Err:       err,
			Parent:    pl.Parent,
			Directory: pl.Directory,
		}))
		return
	}

	if pl.Directory.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("Directory deleted",
		fmt.Sprintf("Directory %s deleted", pl.Directory.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleDeleteDirectoryFailure(evt event.Event) {
	pl := evt.Payload().(directory.DeleteFailed)
	if err := pl.Parent.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if pl.Directory.Is(v.selectedDirectory) {
		v.isSelectedDirLoading.Set(false) // nolint:errcheck
	}

	err := fmt.Errorf("error deleting directory: %w", pl.Err)
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) DeleteFile(file *directory.File) {
	dirNode, err := v.state.Explorer().GetDirectoryNode(file.DirectoryPath())
	if err != nil {
		panic(fmt.Errorf("failed deleting file: %w", err))
	}

	parent := dirNode.Directory()
	evt, err := parent.RemoveFile(file.Name())
	if err != nil {
		return
	}
	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleDeleteFileSuccess(evt event.Event) {
	pl := evt.Payload().(directory.DeleteFileSucceeded)

	if err := pl.ParentDirectory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.state.Explorer().RemoveNode(pl.File.FullPath()); err != nil {
		v.bus.Publish(evt.NewFollowup(directory.DeleteFileFailed{
			Err:             err,
			ParentDirectory: pl.ParentDirectory,
		}))
		return
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File deleted",
		fmt.Sprintf("File %s deleted", pl.File.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleDeleteFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.DeleteFileFailed)
	if err := pl.ParentDirectory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error deleting file: %w", pl.Err)
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
	pl := evt.Payload().(directory.CreateSucceeded)
	if err := v.state.Explorer().PrependDirectory(pl.Directory); err != nil {
		v.bus.Publish(
			evt.NewFollowup(directory.CreateFailed{
				Err:             err,
				ParentDirectory: pl.ParentDirectory,
			}))
		return
	}
	if err := pl.ParentDirectory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
	}
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleCreateDirFailure(evt event.Event) {
	pl := evt.Payload().(directory.CreateFailed)
	if err := pl.ParentDirectory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error creating directory: %w", pl.Err)
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
	pl := evt.Payload().(directory.CreateFileSucceeded)
	if err := v.state.Explorer().UpdateOrAppendFile(pl.File); err != nil {
		v.bus.Publish(evt.NewFollowup(directory.CreateFileFailed{
			Err:       err,
			Directory: pl.Directory,
		}))
		return
	}
	if err := pl.Directory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleCreateFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.CreateFileFailed)
	err := fmt.Errorf("error creating file: %w", pl.Err)
	if notifErr := pl.Directory.Notify(evt); notifErr != nil {
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
	pl := evt.Payload().(directory.RenameSucceeded)
	dir := pl.Directory

	defer func() {
		if dir.Is(v.selectedDirectory) {
			v.isSelectedDirLoading.Set(false) // nolint:errcheck
		}
	}()

	oldPath := dir.Path().String()

	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.state.Explorer().RemoveNode(oldPath); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error removing old directory node: %w", err))
		return
	}

	if err := v.state.Explorer().UpdateOrPrepend(dir); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error updating directory node: %w", err))
		return
	}

	if err := v.LoadDirectory(dir); err != nil {
		v.notifier.NotifyError(fmt.Errorf("error loading the renamed directory: %w", err))
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("Directory renamed",
		fmt.Sprintf("Directory %s renamed to %s", oldPath, dir.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleRenameDirectoryFailure(evt event.Event) {
	pl := evt.Payload().(directory.RenameFailed)
	dir := pl.Directory

	defer func() {
		if dir.Is(v.selectedDirectory) {
			v.isSelectedDirLoading.Set(false) // nolint:errcheck
		}
	}()

	err := fmt.Errorf("error renaming directory: %w", pl.Err)
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
	pl := evt.Payload().(directory.RenameFileSucceeded)
	file := pl.File
	parentDir := pl.Directory

	oldFullPath := file.FullPath()

	if err := parentDir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	if err := v.state.Explorer().RemoveNode(oldFullPath); err != nil {
		v.notifier.NotifyError(fmt.Errorf("handle rename success: %w", err))
		return
	}

	if err := v.state.Explorer().AppendFile(file); err != nil {
		v.notifier.NotifyError(fmt.Errorf("handle rename success: %w", err))
		return
	}

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File renamed",
		fmt.Sprintf("File renamed to %s", file.Name())))
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) handleRenameFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.RenameFileFailed)
	err := fmt.Errorf("error renaming file: %w", pl.Err)
	if err := pl.Directory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.notifier.NotifyError(err)
	v.errorMessage.Set(err.Error()) //nolint:errcheck
	v.triggerStateListeners()
}

func (v *explorerViewModelImpl) initializeTreeData(c *connection_deck.Connection) error {
	if c == nil {
		err := ErrNoConnectionSelected
		v.notifier.NotifyError(err)
		return err
	}

	rootDir, err := directory.NewRoot(c.ID())
	if err != nil {
		newErr := fmt.Errorf("error initializing the root directory: %w", err)
		v.notifier.NotifyError(newErr)
		return newErr
	}

	if err := v.state.Explorer().InitFileTree(rootDir, c.Bucket()); err != nil {
		v.notifier.NotifyError(err)
		return err
	}

	if err := v.LoadDirectory(rootDir); err != nil {
		newErr := fmt.Errorf("error loading root directory: %w", err)
		v.notifier.NotifyError(newErr)
		return newErr
	}

	return nil
}

func (v *explorerViewModelImpl) handleConnectionChange(evt event.Event) {
	var conn *connection_deck.Connection
	pl, ok := evt.Payload().(connection_deck.SelectConnectionSucceeded)
	if ok {
		conn = pl.Connection()
	} else {
		e := evt.Payload().(connection_deck.UpdateConnectionSucceeded)
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
	pl := evt.Payload().(connection_deck.RemoveConnectionSucceeded)
	conn := pl.Connection()
	if v.selectedConnectionVal != nil && v.selectedConnectionVal.Is(conn) {
		v.selectedConnectionVal = nil
		v.selectedConnection.Set(nil) //nolint:errcheck
	}
}
