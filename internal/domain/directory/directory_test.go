package directory_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func TestDirectory(t *testing.T) {
	t.Run("should change directory states", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

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
		dir.SetLoaded(true)
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
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

		// When
		dir.Load() //nolint:errcheck
		dir.SetLoaded(false)

		// Then
		assert.False(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsOpened())
	})
}

func TestDirectory_Load(t *testing.T) {
	t.Run("should load then update directory content on success", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)
		require.False(t, dir.IsLoaded())

		d1, _ := directory.New(connection_deck.NewConnectionID(), "data/d1", dir.Path())
		d2, _ := directory.New(connection_deck.NewConnectionID(), "data/d2", dir.Path())
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

		resSubDirs, _ := dir.SubDirectories()
		assert.Len(t, resSubDirs, 2)
		resFiles, _ := dir.Files()
		assert.Len(t, resFiles, 2)
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
	})

	t.Run("should return error when loading is already in progress", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

		_, err = dir.Load()
		require.NoError(t, err)

		// When
		_, err = dir.Load()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "loading is still in progress")
		assert.True(t, dir.IsLoading())
		assert.False(t, dir.IsLoaded())
	})

	t.Run("should return error when directory is already loaded", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

		_, err = dir.Load()
		require.NoError(t, err)
		dir.SetLoaded(true)
		require.True(t, dir.IsLoaded())

		// When
		_, err = dir.Load()

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already loaded")
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
		assert.False(t, dir.IsOpened())
	})

	t.Run("should return error when directory is already opened", func(t *testing.T) {
		// Given
		dir, err := directory.New(connection_deck.NewConnectionID(), "data", directory.RootPath)
		require.NoError(t, err)

		_, err = dir.Load()
		require.NoError(t, err)
		dir.SetLoaded(true)
		dir.Open()
		require.True(t, dir.IsOpened())

		// When
		_, err = dir.Load()

		// Then
		assert.Error(t, err)
		assert.True(t, dir.IsOpened())
		assert.True(t, dir.IsLoaded())
		assert.False(t, dir.IsLoading())
	})
}

func TestDirectory_RemoveFile(t *testing.T) {
	t.Run("should create file deleted event when file exists", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		dir, err := directory.New(connID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		f1, _ := directory.NewFile("main.go", dir.Path())
		f2, _ := directory.NewFile("readme.md", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{f1, f2})

		_, err = dir.Load()
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
		resFiles, _ := dir.Files()
		assert.Len(t, resFiles, 1)
	})

	t.Run("should return error when file does not exist", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		dir, err := directory.New(connID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err = dir.Load()
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
		connID := connection_deck.NewConnectionID()
		dir, err := directory.New(connID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		f1, _ := directory.NewFile("main.go", dir.Path())
		f2, _ := directory.NewFile("readme.md", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, nil, []*directory.File{f1, f2})

		_, err = dir.Load()
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
		resFiles, _ := dir.Files()
		assert.Len(t, resFiles, 2)
	})
}

func TestDirectory_RemoveSubDirectory(t *testing.T) {
	t.Run("should create directory deleted event when subdirectory exists", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		dir, err := directory.New(connID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		subDir1, _ := directory.New(connID, "sub1", dir.Path())
		subDir2, _ := directory.New(connID, "sub2", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, []*directory.Directory{subDir1, subDir2}, nil)

		_, err = dir.Load()
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
		resSubDirs, _ := dir.SubDirectories()
		assert.Len(t, resSubDirs, 1)
	})

	t.Run("should return error when subdirectory does not exist", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		dir, err := directory.New(connID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		loadEvt := directory.NewLoadSuccessEvent(dir, nil, nil)
		_, err = dir.Load()
		require.NoError(t, err)
		require.NoError(t, dir.Notify(loadEvt))

		// When
		_, err = dir.RemoveSubDirectory("missing")

		// Then
		assert.ErrorIs(t, err, directory.ErrNotFound)
	})

	t.Run("shouldn't remove the subdirectory when a failure event is emitted", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()
		dir, err := directory.New(connID, directory.RootDirName, directory.NilParentPath)
		require.NoError(t, err)

		subDir1, _ := directory.New(connID, "sub1", dir.Path())
		subDir2, _ := directory.New(connID, "sub2", dir.Path())
		loadEvt := directory.NewLoadSuccessEvent(dir, []*directory.Directory{subDir1, subDir2}, nil)

		_, err = dir.Load()
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
		resSubDirs, _ := dir.SubDirectories()
		assert.Len(t, resSubDirs, 2)
	})
}
