package directory_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestDirectory(t *testing.T) {
	t.Run("should change directory states", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "data", directory.RootPath)

		// When & Then
		// not loaded directory
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsOpened())

		// loading it
		evt, err := dir.Load()
		assert.NoError(t, err)
		assert.Equal(t, directory.LoadEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())
		assert.True(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsOpened())

		// loading ended sucssesffuly
		require.NoError(t, dir.Notify(directory.NewLoadSuccessEvent(dir, nil, nil)))
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsOpened())

		// Then, open it
		dir.Open()
		assert.NoError(t, err)
		assert.True(t, dir.IsOpened())
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
	})

	t.Run("should change directory states with error", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "data", directory.RootPath)

		// When
		dir.Load() //nolint:errcheck
		require.NoError(t, dir.Notify(directory.NewLoadFailureEvent(errors.New("ckc"), dir)))

		// Then
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsOpened())
	})
}

func TestDirectory_Load(t *testing.T) {
	t.Run("should load then update directory content on success", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "data", directory.RootPath)
		require.False(t, dir.IsLoaded())

		d1, _ := directory.New(connection_deck.NewConnectionID(), "data/d1", dir)
		d2, _ := directory.New(connection_deck.NewConnectionID(), "data/d2", dir)
		subDirs := []*directory.Directory{
			d1, d2,
		}

		f1, _ := directory.NewFile("main.go", dir.Path())
		f2, _ := directory.NewFile("readme.md", dir.Path())
		files := []*directory.File{
			f1, f2,
		}

		successEvt := directory.NewLoadSuccessEvent(dir, subDirs, files)

		// When & Then
		evt, err := dir.Load()
		assert.NoError(t, err)
		assert.Equal(t, directory.LoadEventType, evt.Type())
		assert.True(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())

		err = dir.Notify(successEvt)
		assert.NoError(t, err)

		resSubDirs := dir.SubDirectories()
		assert.Len(t, resSubDirs, 2)
		resFiles := dir.Files()
		assert.Len(t, resFiles, 2)
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
	})

	t.Run("should return error when loading is already in progress", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "data", directory.RootPath)

		_, err := dir.Load()
		require.NoError(t, err)

		// When
		_, err = dir.Load()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "loading is still in progress")
		assert.True(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())
	})

	t.Run("should trigger a reload when directory is already loaded", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "data", directory.RootPath)

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(directory.NewLoadSuccessEvent(dir, nil, nil)))
		require.True(t, dir.IsLoaded())

		// When
		res, err := dir.Load()

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.LoadEventType, res.Type())
		assert.Equal(t, dir, res.Directory())
	})
}

func TestDirectory_NewFile(t *testing.T) {
	t.Run("should create a file and add it to the directory on success", func(t *testing.T) {
		// Given
		dir := testutil.FakeLoadedRootDirectory(t)

		// When
		evt, err := dir.NewFile("report.csv", false)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.FileCreatedEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())
		assert.Equal(t, "report.csv", evt.File().Name().String())
		files := dir.Files()
		require.Len(t, files, 0)

		// Then, when notified of the success
		successEvt := directory.NewFileCreatedSuccessEvent(dir, evt.File())
		assert.NoError(t, dir.Notify(successEvt))

		files = dir.Files()
		require.Len(t, files, 1)
		assert.Equal(t, "report.csv", files[0].Name().String())
	})
}

func TestDirectory_RemoveFile(t *testing.T) {
	t.Run("should create file deleted event when file exists, then recreated", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		f1, _ := directory.NewFile("main.go", dir.Path())
		f2, _ := directory.NewFile("readme.md", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{f1, f2})

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		successEvt := directory.NewFileDeletedSuccessEvent(dir, f1)

		// When
		evt, err := dir.RemoveFile(f1.Name())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.FileDeletedEventType, evt.Type())
		assert.Equal(t, dir, evt.Parent())
		assert.Equal(t, f1, evt.File())

		assert.NoError(t, dir.Notify(successEvt))
		resFiles := dir.Files()
		assert.Len(t, resFiles, 1)

		// Then, we recreate the deleted file
		newFileEvt, err := dir.NewFile("main.go", false)
		assert.NoError(t, err)

		newFileSuccessEvt := directory.NewFileCreatedSuccessEvent(dir, newFileEvt.File())
		assert.NoError(t, dir.Notify(newFileSuccessEvt))

		resFiles = dir.Files()
		assert.Len(t, resFiles, 2)
		assert.Equal(t, "main.go", resFiles[1].Name().String())
	})

	t.Run("should emit file deleted event when file exists, then re-uploaded", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		f1, _ := directory.NewFile("main.go", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{f1})

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		successEvt := directory.NewFileDeletedSuccessEvent(dir, f1)

		// When
		evt, err := dir.RemoveFile(f1.Name())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.FileDeletedEventType, evt.Type())
		assert.Equal(t, dir, evt.Parent())
		assert.Equal(t, f1, evt.File())

		assert.NoError(t, dir.Notify(successEvt))
		resFiles := dir.Files()
		assert.Len(t, resFiles, 0)

		// Then, we upload again the deleted file
		uploadFileEvt, err := dir.UploadFile("project/src/main.go", false)
		assert.NoError(t, err)

		uploadedSuccessFileEvt := directory.NewContentUploadedSuccessEvent(dir, uploadFileEvt.Content().File())
		assert.NoError(t, dir.Notify(uploadedSuccessFileEvt))

		assert.Len(t, dir.Files(), 1)
		assert.Equal(t, "main.go", dir.Files()[0].Name().String())
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		missingName, _ := directory.NewFileName("missing.txt")
		_, err = dir.RemoveFile(missingName)

		// Then
		assert.ErrorIs(t, err, directory.ErrNotFound)
	})

	t.Run("shouldn't remove the file when a failure event is emitted", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		f1, _ := directory.NewFile("main.go", dir.Path())
		f2, _ := directory.NewFile("readme.md", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{f1, f2})

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		failureEvt := directory.NewFileDeletedFailureEvent(
			errors.New("ckc"), dir)

		// When
		evt, err := dir.RemoveFile(f1.Name())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.FileDeletedEventType, evt.Type())
		assert.Equal(t, dir, evt.Parent())

		assert.NoError(t, dir.Notify(failureEvt))
		resFiles := dir.Files()
		assert.Len(t, resFiles, 2)
	})
}

func TestDirectory_RemoveSubDirectory(t *testing.T) {
	t.Run("should create directory deleted event when subdirectory exists", func(t *testing.T) {
		// Given
		connID := testutil.FakeS3LikeConnectionId
		dir := testutil.FakeNotLoadedRootDirectory(t)

		subDir1, _ := directory.New(connID, "sub1", dir)
		subDir2, _ := directory.New(connID, "sub2", dir)
		loadEvt := directory.NewLoadSuccessEvent(dir, []*directory.Directory{subDir1, subDir2}, nil)

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		successEvt := directory.NewDeletedSuccessEvent(subDir1)

		// When
		evt, err := dir.RemoveSubDirectory("sub1")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.DeletedEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())
		assert.Equal(t, subDir1.Path(), evt.DeletedDirPath())

		assert.NoError(t, dir.Notify(successEvt))
		resSubDirs := dir.SubDirectories()
		assert.Len(t, resSubDirs, 1)
	})

	t.Run("should return error when subdirectory does not exist", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		_, err = dir.RemoveSubDirectory("missing")

		// Then
		assert.ErrorIs(t, err, directory.ErrNotFound)
	})

	t.Run("shouldn't remove the subdirectory when a failure event is emitted", func(t *testing.T) {
		// Given
		connID := testutil.FakeS3LikeConnectionId
		dir := testutil.FakeNotLoadedRootDirectory(t)

		subDir1, _ := directory.New(connID, "sub1", dir)
		subDir2, _ := directory.New(connID, "sub2", dir)
		loadEvt := directory.NewLoadSuccessEvent(dir, []*directory.Directory{subDir1, subDir2}, nil)

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		failureEvt := directory.NewDeletedFailureEvent(errors.New("ckc"))

		// When
		evt, err := dir.RemoveSubDirectory("sub1")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.DeletedEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())

		assert.NoError(t, dir.Notify(failureEvt))
		resSubDirs := dir.SubDirectories()
		assert.Len(t, resSubDirs, 2)
	})
}

func TestDirectory_UploadFile(t *testing.T) {
	t.Run("should emit upload event and add file on success", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		evt, err := dir.UploadFile("local/report.csv", false)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.ContentUploadedEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())
		assert.Equal(t, "report.csv", evt.Content().File().Name().String())

		file, _ := directory.NewFile("report.csv", dir.Path())
		successEvt := directory.NewContentUploadedSuccessEvent(dir, file)
		assert.NoError(t, dir.Notify(successEvt))

		files := dir.Files()
		require.Len(t, files, 1)
		assert.Equal(t, "report.csv", files[0].Name().String())
	})

	t.Run("should emit upload event and add file on success on a non empty directory", func(t *testing.T) {
		// Given
		connID := testutil.FakeS3LikeConnectionId
		dir := testutil.NewNotLoadedDirectory(t, "data", directory.RootPath)

		d1, _ := directory.New(connID, "d1", dir)
		d2, _ := directory.New(connID, "d2", dir)
		subDirs := []*directory.Directory{d1, d2}
		f1, _ := directory.NewFile("main.go", dir.Path())
		f2, _ := directory.NewFile("readme.md", dir.Path())
		files := []*directory.File{f1, f2}

		loadEvt := directory.NewLoadSuccessEvent(dir, subDirs, files)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		evt, err := dir.UploadFile("local/report.csv", false)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.ContentUploadedEventType, evt.Type())
		assert.Equal(t, dir, evt.Directory())
		assert.Equal(t, "report.csv", evt.Content().File().Name().String())

		file, _ := directory.NewFile("report.csv", dir.Path())
		successEvt := directory.NewContentUploadedSuccessEvent(dir, file)
		assert.NoError(t, dir.Notify(successEvt))

		resFiles := dir.Files()
		require.Len(t, resFiles, 3)
		assert.Equal(t, "main.go", resFiles[0].Name().String())
		assert.Equal(t, "readme.md", resFiles[1].Name().String())
		assert.Equal(t, "report.csv", resFiles[2].Name().String())
	})

	t.Run("should overwrite existing file when upload succeeds", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		existing, _ := directory.NewFile("report.csv", dir.Path(), directory.WithFileSize(42))
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{existing})
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		evt, err := dir.UploadFile("tmp/report.csv", true)

		// Then
		assert.NoError(t, err)

		file, _ := directory.NewFile("report.csv", dir.Path(), directory.WithFileSize(1337))
		successEvt := directory.NewContentUploadedSuccessEvent(dir, file)
		assert.NoError(t, dir.Notify(successEvt))

		assert.Equal(t, "report.csv", evt.Content().File().Name().String())

		files := dir.Files()
		require.Len(t, files, 1)
		assert.True(t, files[0].Equal(file))
		assert.Equal(t, 1337, files[0].SizeBytes())
		assert.Equal(t, "report.csv", files[0].Name().String())
	})

	t.Run("should return an error when the file already exists remotely in the directory", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		existing, _ := directory.NewFile("report.csv", dir.Path(), directory.WithFileSize(42))
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{existing})
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		_, err = dir.UploadFile("tmp/report.csv", false)

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, directory.ErrAlreadyExists)
		assert.Contains(t, err.Error(), "file report.csv already exists in directory /")
	})

	t.Run("should return error when directory is not loaded", func(t *testing.T) {
		// Given
		dir := testutil.FakeNotLoadedRootDirectory(t)

		// When
		_, err := dir.UploadFile("local/report.csv", false)

		// Then
		assert.ErrorIs(t, err, directory.ErrNotLoaded)
	})
}

func TestDirectory_Rename(t *testing.T) {
	t.Run("should emit event and not yet rename the directory", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "oldname", "/parent/")
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)

		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		evt, err := dir.Rename("newname")

		// Then
		require.NoError(t, err)
		assert.Equal(t, directory.RenameEventType, evt.Type())
		assert.Equal(t, "oldname", dir.Name())
		assert.Equal(t, directory.Path("/parent/oldname/"), dir.Path())
		assert.Equal(t, "newname", evt.NewName())
	})

	t.Run("should return error when directory is not loaded", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "oldname", "/parent/")

		// When
		_, err := dir.Rename("newname")

		// Then
		assert.ErrorIs(t, err, directory.ErrNotLoaded)
	})

	t.Run("should return error when new name is invalid", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "oldname", "/parent/")

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When & Then - empty name
		_, err = dir.Rename("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory name is empty")

		// When & Then - name with slash
		_, err = dir.Rename("new/name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory name should not contain '/'s")

		// When & Then - name is just slash
		_, err = dir.Rename("/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory name should not be '/'")
	})

	t.Run("should return error when trying to rename to same name", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "oldname", "/parent/")

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When - try to rename to the same name
		_, err = dir.Rename("oldname")

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "new name must be different from current name")
	})

	t.Run("should update directory state on rename success event", func(t *testing.T) {
		// Given
		dir := testutil.NewNotLoadedDirectory(t, "oldname", "/parent/")

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err := dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		_, err = dir.Rename("newname")
		require.NoError(t, err)

		// When
		successEvt := directory.NewRenameSuccessEvent(dir, "newname")
		err = dir.Notify(successEvt)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "newname", dir.Name())
		assert.Equal(t, directory.Path("/parent/newname/"), dir.Path())
	})

	t.Run("should update sub directories' parent path recursively", func(t *testing.T) {
		// Given
		dir := testutil.NewLoadedDirectory(t, "oldname", directory.RootPath)
		f1 := testutil.AddFileToDirectory(t, dir, "f1.txt")

		subdir1 := testutil.AddSubDirectoryToDirectory(t, dir, "sub1")
		f11 := testutil.AddFileToDirectory(t, subdir1, "f11.txt")

		subdir2 := testutil.AddSubDirectoryToDirectory(t, dir, "sub2")

		subSubDir1 := testutil.AddSubDirectoryToDirectory(t, subdir1, "subsub1")
		f111 := testutil.AddFileToDirectory(t, subSubDir1, "f111.txt")

		subSubDir2NoLoaded := testutil.AddSubNotLoadedDirectoryToDirectory(t, subdir2, "subsub2")

		require.Equal(t, directory.Path("/oldname/"), dir.Path())
		require.Equal(t, directory.Path("/oldname/sub1/"), subdir1.Path())
		require.Equal(t, directory.Path("/oldname/sub2/"), subdir2.Path())
		require.Equal(t, directory.Path("/oldname/sub1/subsub1/"), subSubDir1.Path())
		require.Equal(t, directory.Path("/oldname/sub2/subsub2/"), subSubDir2NoLoaded.Path())

		require.Equal(t, "/oldname/f1.txt", f1.FullPath())
		require.Equal(t, "/oldname/sub1/f11.txt", f11.FullPath())
		require.Equal(t, "/oldname/sub1/subsub1/f111.txt", f111.FullPath())

		_, err := dir.Rename("newname")
		require.NoError(t, err)

		// When
		successEvt := directory.NewRenameSuccessEvent(dir, "newname")
		require.NoError(t, dir.Notify(successEvt))

		// Then
		assert.Equal(t, directory.Path("/newname/"), dir.Path())
		assert.Equal(t, directory.Path("/newname/sub1/"), subdir1.Path())
		assert.Equal(t, directory.Path("/newname/sub2/"), subdir2.Path())
		assert.Equal(t, directory.Path("/newname/sub1/subsub1/"), subSubDir1.Path())
		assert.Equal(t, directory.Path("/newname/sub2/subsub2/"), subSubDir2NoLoaded.Path())

		assert.Equal(t, "/newname/f1.txt", f1.FullPath())
		assert.Equal(t, "/newname/sub1/f11.txt", f11.FullPath())
		assert.Equal(t, "/newname/sub1/subsub1/f111.txt", f111.FullPath())
	})
}

func TestDirectory_Resume(t *testing.T) {
	t.Run("should returns a rename resume event when directory is in a resumable state with a rename pending status after a failing rename", func(t *testing.T) {
		// Given
		root := testutil.FakeLoadedRootDirectory(t)
		oldDir := testutil.AddSubNotLoadedDirectoryToDirectory(t, root, "oldname")
		newDir := testutil.AddSubNotLoadedDirectoryToDirectory(t, root, "newname")

		_, err := oldDir.Load()
		require.NoError(t, err)

		urErr := directory.UncompletedRename{
			SourceDirPath:      "/oldname/",
			DestinationDirPath: "/newname/",
		}
		loadFailureEvent := directory.NewLoadFailureEvent(urErr, oldDir) // Simulate a loading failure due to an inconsistent state from a rename failure
		require.NoError(t, oldDir.Notify(loadFailureEvent))

		require.Equal(t, directory.RenameFailedStatus{
			CurrentDirectory: oldDir,
			IsSourceDir:      true,
			OtherDirPath:     "/newname/",
		}, oldDir.Status())

		// When
		evt, err := oldDir.Recover(directory.RecoveryChoiceRenameResume)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.RenameRecoverEventType, evt.Type())

		res := evt.(directory.RenameRecoverEvent)
		assert.Equal(t, newDir, res.DstDir())
		assert.Equal(t, oldDir, res.Directory())
		assert.Equal(t, directory.RecoveryChoiceRenameResume, res.Choice())

		assert.True(t, oldDir.IsLoading())
		assert.True(t, newDir.IsLoading())
	})

	t.Run("should returns a rename resume event when directory is in a resumable state with a rename pending status after a failing loading", func(t *testing.T) {
		// Given
		root := testutil.FakeLoadedRootDirectory(t)
		oldDir := testutil.AddSubNotLoadedDirectoryToDirectory(t, root, "oldname")
		newDir := testutil.AddSubNotLoadedDirectoryToDirectory(t, root, "newname")

		urErr := directory.UncompletedRename{
			SourceDirPath:      "/oldname/",
			DestinationDirPath: "/newname/",
		}
		_, err := oldDir.Load()
		require.NoError(t, err)
		require.NoError(t, oldDir.Notify(directory.NewLoadFailureEvent(urErr, oldDir)))

		// When
		evt, err := oldDir.Recover(directory.RecoveryChoiceRenameResume)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.RenameFailedStatus{
			CurrentDirectory: newDir,
			IsSourceDir:      false,
			OtherDirPath:     "/oldname/",
		}, newDir.Status())
		assert.Equal(t, directory.RenameRecoverEventType, evt.Type())

		res := evt.(directory.RenameRecoverEvent)
		assert.Equal(t, newDir, res.DstDir())
		assert.Equal(t, oldDir, res.Directory())
	})
}
