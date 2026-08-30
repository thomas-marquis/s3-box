package viewmodel

import (
	"errors"
	"io/fs"
	"os"
	"sync"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/s3box"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
	"github.com/thomas-marquis/s3-box/internal/ui/values"

	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

// ExplorerViewModel represents the view model for the file explorer interface.
// It handles the tree structure display, file operations, and directory management
// while maintaining the connection with the underlying storage system.
type ExplorerViewModel interface {
	ViewModel

	// LoadDirectory sync a directory with the actual s3 one and load its files and children.
	// If the directory is already open, it will do nothing.
	LoadDirectory(dir *directory.Directory) error

	ReloadDirectory(dir *directory.Directory) error

	// DownloadFile downloads a file to the specified local destination
	DownloadFile(f *directory.File, dest string)

	DownloadDirectory(dir *directory.Directory, dest string)

	PrepareUpload(uris []fyne.URI, dir *directory.Directory) error
	DoUpload(localBasePath string, preview *directory.Preview, strategy directory.MaterializeStrategy)
	UploadOne(localPath string, dir *directory.Directory, overwrite bool) error

	// DeleteFile removes a file from storage and updates the tree
	DeleteFile(file *directory.File)

	DeleteDirectory(dir *directory.Directory)

	// CreateEmptyDirectory creates an empty subdirectory in the given parent directory
	CreateEmptyDirectory(parent *directory.Directory, name string)

	// CreateEmptyFile creates an empty file in the given parent directory
	CreateEmptyFile(parent *directory.Directory, name string)

	// RenameDirectory renames a directory
	RenameDirectory(dir *directory.Directory, newName string)

	// RenameFile renames a file
	RenameFile(file *directory.File, newName string)

	ResumeRename(dir *directory.Directory) error
	RollbackRename(dir *directory.Directory) error
	AbortRename(dir *directory.Directory) error
}

type explorerViewModelImpl struct {
	baseViewModel
	sync.Mutex

	notifier notification.Repository
	bus      event.Bus
	state    *state.State
}

func NewExplorerViewModel(
	notifier notification.Repository,
	bus event.Bus,
	st *state.State,
) ExplorerViewModel {
	v := &explorerViewModelImpl{
		baseViewModel: baseViewModel{
			errorMessage: binding.NewString(),
			infoMessage:  binding.NewString(),
		},
		notifier: notifier,
		bus:      bus,
		state:    st,
	}

	st.Connection().Selected().AddListener(binding.NewDataListener(func() {
		newSelected := u.SkipV(st.Connection().Selected().Get())
		if newSelected == nil {
			st.Explorer().ResetTree()
			return
		}

		if err := v.initializeTreeData(newSelected); err != nil {
			u.Skip(v.errorMessage.Set(err.Error()))
			return
		}
	}))

	bus.Subscribe().
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
		On(event.Is(s3box.UserValidationRefusedType), v.handleUserValidationRefused).
		On(event.Is(directory.UploadReadyType), v.handleUploadReady).
		On(event.Is(directory.DeleteFailedType), v.handleDeleteDirectoryFailure).
		On(event.Is(directory.DeleteSucceededType), v.handleDeleteDirectorySuccess).
		On(event.Is(directory.DownloadSucceededType), v.handleDownloadSuccess).
		On(event.Is(directory.DownloadFailedType), v.handleDownloadFailure).
		ListenWithWorkers(values.ExplorerNbWorkers)

	return v
}

func (v *explorerViewModelImpl) handleUserValidationRefused(evt event.Event) {
	pl := evt.Payload().(s3box.UserValidationRefused)
	reason, ok := pl.Reason.Payload().(directory.RenameTriggered)
	if !ok {
		return
	}

	if err := reason.Directory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
}

func (v *explorerViewModelImpl) LoadDirectory(dir *directory.Directory) error {
	if !v.state.Connection().HasSelected() {
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

	return nil
}

func (v *explorerViewModelImpl) ReloadDirectory(dir *directory.Directory) error {
	if !v.state.Connection().HasSelected() {
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
}

func (v *explorerViewModelImpl) handleLoadDirFailure(evt event.Event) {
	pl := evt.Payload().(directory.LoadFailed)
	dir := pl.Directory
	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	u.Skip(v.infoMessage.Set(pl.Err.Error()))
}

func (v *explorerViewModelImpl) DownloadFile(f *directory.File, dest string) {
	evt := f.Download(u.SkipV(v.state.Connection().Selected().Get()).ID(), dest)

	dirPath := filepath.Dir(dest)
	uriLister := uu.ToListableURI(dirPath)
	fyne.CurrentApp().Preferences().SetString(values.PrefDownloadLocation, uriLister.Path())
	u.Skip(v.state.Explorer().DownloadLocation().Set(uriLister))

	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleDownloadFileSuccess(evt event.Event) {
	pl := evt.Payload().(directory.DownloadFileSucceeded)
	u.Skip(v.infoMessage.Set(
		fmt.Sprintf("File %s downloaded", pl.File.Name())))
}

func (v *explorerViewModelImpl) handleDownloadFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.DownloadFileFailed)
	err := fmt.Errorf("error downloading file: %w", pl.Err)
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) DownloadDirectory(dir *directory.Directory, destParent string) {
	uriLister := uu.ToListableURI(destParent)
	fyne.CurrentApp().Preferences().SetString(values.PrefDownloadLocation, uriLister.Path())
	u.Skip(v.state.Explorer().DownloadLocation().Set(uriLister))

	v.bus.Publish(event.New(directory.DownloadTriggered{
		Directory:      dir,
		DestParentPath: destParent,
	}))
}

func (v *explorerViewModelImpl) handleDownloadSuccess(evt event.Event) {
	pl := evt.Payload().(directory.DownloadSucceeded)
	if err := pl.Directory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		u.Skip(v.errorMessage.Set(err.Error()))
		return
	}
	u.Skip(v.infoMessage.Set(
		fmt.Sprintf("Directory %s downloaded successfully", pl.Directory.Name())))
}

func (v *explorerViewModelImpl) handleDownloadFailure(evt event.Event) {
	pl := evt.Payload().(directory.DownloadFailed)
	err := fmt.Errorf("error downloading file: %w", pl.Err)
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) DoUpload(localBasePath string, preview *directory.Preview, strategy directory.MaterializeStrategy) {
	uploadMat := directory.NewUploadMaterializer(preview, localBasePath)
	v.bus.Publish(uploadMat.Materialize(strategy))
}

func (v *explorerViewModelImpl) UploadOne(localPath string, dir *directory.Directory, overwrite bool) error {
	if !v.state.Connection().HasSelected() {
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

	uriLister := uu.ToListableURI(filepath.Dir(localPath))
	fyne.CurrentApp().Preferences().SetString(values.PrefUploadLocation, uriLister.Path())
	u.Skip(v.state.Explorer().UploadLocation().Set(uriLister))

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
}

func (v *explorerViewModelImpl) handleFileUploadFailure(evt event.Event) {
	pl := evt.Payload().(directory.UploadFileFailed)
	err := fmt.Errorf("error uploading file: %w", pl.Err)
	if notifErr := pl.Directory.Notify(evt); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) PrepareUpload(uris []fyne.URI, dir *directory.Directory) error {
	prev, err := makePreviewFromUris(uu.FromFyneUrisToPaths(uris), dir)
	if err != nil {
		return err
	}

	loadMat := directory.NewLoadMaterializer(prev, directory.UploadReady{
		Directory: dir,
		SrcPaths:  uu.FromFyneUrisToPaths(uris),
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

	localParentDirUri := uu.GetCommonParentPath(pl.SrcPaths)

	prev, err := makePreviewFromUris(pl.SrcPaths, pl.Directory)
	if err != nil {
		v.notifier.NotifyError(err)
		return
	}

	u.Skip(v.state.Explorer().UploadPreview().Set(state.UploadPreviewState{
		Preview: prev,
		BaseUri: localParentDirUri,
	}))
}

func (v *explorerViewModelImpl) DeleteDirectory(dir *directory.Directory) {
	if directory.RootPath.Is(dir) {
		u.Skip(v.errorMessage.Set("Cannot delete root directory"))
		return
	}

	parent := dir.Parent()

	evt, err := parent.RemoveSubDirectory(dir.Name())
	if err != nil {
		u.Skip(v.errorMessage.Set(err.Error()))
		return
	}

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

	v.state.Explorer().SelectDir(pl.Directory.Parent())

	fyne.CurrentApp().SendNotification(fyne.NewNotification("Directory deleted",
		fmt.Sprintf("Directory %s deleted", pl.Directory.Name())))
}

func (v *explorerViewModelImpl) handleDeleteDirectoryFailure(evt event.Event) {
	pl := evt.Payload().(directory.DeleteFailed)
	if err := pl.Parent.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}

	err := fmt.Errorf("error deleting directory: %w", pl.Err)
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
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

	v.state.Explorer().SelectDir(pl.ParentDirectory)

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File deleted",
		fmt.Sprintf("File %s deleted", pl.File.Name())))
}

func (v *explorerViewModelImpl) handleDeleteFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.DeleteFileFailed)
	if err := pl.ParentDirectory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error deleting file: %w", pl.Err)
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) CreateEmptyDirectory(parent *directory.Directory, name string) {
	if !v.state.Connection().HasSelected() {
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
}

func (v *explorerViewModelImpl) handleCreateDirFailure(evt event.Event) {
	pl := evt.Payload().(directory.CreateFailed)
	if err := pl.ParentDirectory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	err := fmt.Errorf("error creating directory: %w", pl.Err)
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) CreateEmptyFile(parent *directory.Directory, name string) {
	if !v.state.Connection().HasSelected() {
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
}

func (v *explorerViewModelImpl) handleCreateFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.CreateFileFailed)
	err := fmt.Errorf("error creating file: %w", pl.Err)
	if notifErr := pl.Directory.Notify(evt); notifErr != nil {
		err = fmt.Errorf("%w: error notifying parent directory: %w", err, notifErr)
	}
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) RenameDirectory(dir *directory.Directory, newName string) {
	if !v.state.Connection().HasSelected() {
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

	v.bus.Publish(evt)
}

func (v *explorerViewModelImpl) handleRenameDirectorySuccess(evt event.Event) {
	pl := evt.Payload().(directory.RenameSucceeded)
	dir := pl.Directory

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
}

func (v *explorerViewModelImpl) handleRenameDirectoryFailure(evt event.Event) {
	pl := evt.Payload().(directory.RenameFailed)
	dir := pl.Directory

	err := fmt.Errorf("error renaming directory: %w", pl.Err)
	if err := dir.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
}

func (v *explorerViewModelImpl) ResumeRename(dir *directory.Directory) error {
	evt, err := dir.Recover(directory.RecoveryChoiceRenameResume)
	if err != nil {
		return fmt.Errorf("impossible to resume rename: %w", err)
	}
	v.bus.Publish(evt)
	return nil
}

func (v *explorerViewModelImpl) RollbackRename(dir *directory.Directory) error {
	evt, err := dir.Recover(directory.RecoveryChoiceRenameRollback)
	if err != nil {
		return fmt.Errorf("impossible to rollback rename: %w", err)
	}
	v.bus.Publish(evt)
	return nil
}

func (v *explorerViewModelImpl) AbortRename(dir *directory.Directory) error {
	evt, err := dir.Recover(directory.RecoveryChoiceRenameAbort)
	if err != nil {
		return fmt.Errorf("impossible to abort rename: %w", err)
	}
	v.bus.Publish(evt)
	return nil
}

func (v *explorerViewModelImpl) RenameFile(file *directory.File, newName string) {
	if !v.state.Connection().HasSelected() {
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

	v.state.Explorer().SelectFile(file)

	fyne.CurrentApp().SendNotification(fyne.NewNotification("File renamed",
		fmt.Sprintf("File renamed to %s", file.Name())))
}

func (v *explorerViewModelImpl) handleRenameFileFailure(evt event.Event) {
	pl := evt.Payload().(directory.RenameFileFailed)
	err := fmt.Errorf("error renaming file: %w", pl.Err)
	if err := pl.Directory.Notify(evt); err != nil {
		v.notifier.NotifyError(err)
		return
	}
	v.notifier.NotifyError(err)
	u.Skip(v.errorMessage.Set(err.Error()))
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
