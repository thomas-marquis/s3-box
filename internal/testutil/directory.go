package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

// FakeNotLoadedRootDirectory creates a new root directory (not loaded) with FakeS3LikeConnectionId
func FakeNotLoadedRootDirectory(t *testing.T) *directory.Directory {
	t.Helper()

	dir, err := directory.NewRoot(FakeS3LikeConnectionId)
	require.NoError(t, err)

	return dir
}

// FakeLoadedRootDirectory creates a new root directory (loaded) with FakeS3LikeConnectionId
func FakeLoadedRootDirectory(t *testing.T) *directory.Directory {
	t.Helper()

	dir, err := directory.NewRoot(FakeS3LikeConnectionId)
	require.NoError(t, err)

	_, err = dir.Load()
	require.NoError(t, err)

	err = dir.Notify(event.New(directory.LoadSucceeded{
		Directory: dir,
	}))
	require.NoError(t, err)

	return dir
}

// NewLoadedDirectoryWithConn creates a new loaded directory with the provided connection ID
func NewLoadedDirectoryWithConn(t *testing.T, connID connection_deck.ConnectionID, name string, parentPath directory.Path) *directory.Directory {
	t.Helper()

	parent, err := directory.NewRoot(connID)
	require.NoError(t, err)

	if parentPath != directory.RootPath {
		for _, name := range parentPath.Split() {
			dir, err := directory.New(connID, name, parent)
			require.NoError(t, err)

			_, err = dir.Load()
			require.NoError(t, err)

			err = dir.Notify(event.New(directory.LoadSucceeded{
				Directory: dir,
			}))
			require.NoError(t, err)
			parent = dir
		}
	}

	dir, err := directory.New(connID, name, parent)
	require.NoError(t, err)

	_, err = dir.Load()
	require.NoError(t, err)

	err = dir.Notify(event.New(directory.LoadSucceeded{
		Directory: dir,
	}))
	require.NoError(t, err)

	return dir
}

// NewLoadedDirectory creates a new loaded directory with FakeS3LikeConnectionId
func NewLoadedDirectory(t *testing.T, name string, parentPath directory.Path) *directory.Directory {
	t.Helper()
	return NewLoadedDirectoryWithConn(t, FakeS3LikeConnectionId, name, parentPath)
}

// NewNotLoadedDirectoryWithConn creates a new unloaded directory with the provided connection ID, but with loaded parents chain
func NewNotLoadedDirectoryWithConn(t *testing.T, connID connection_deck.ConnectionID, name string, parentPath directory.Path) *directory.Directory {
	t.Helper()

	parent, err := directory.NewRoot(connID)
	require.NoError(t, err)

	if parentPath != directory.RootPath {
		for _, name := range parentPath.Split() {
			dir, err := directory.New(connID, name, parent)
			require.NoError(t, err)

			_, err = dir.Load()
			require.NoError(t, err)

			err = dir.Notify(event.New(directory.LoadSucceeded{
				Directory: dir,
			}))
			require.NoError(t, err)
			parent = dir
		}
	}

	dir, err := directory.New(connID, name, parent)
	require.NoError(t, err)

	return dir
}

// NewNotLoadedDirectory creates a new unloaded directory with FakeS3LikeConnectionId, but with loaded parents chain
func NewNotLoadedDirectory(t *testing.T, name string, parentPath directory.Path) *directory.Directory {
	t.Helper()
	return NewNotLoadedDirectoryWithConn(t, FakeS3LikeConnectionId, name, parentPath)
}

// AddFileToDirectory creates a new file in the provided directory, then returns the new file.
// The connection id used is FakeS3LikeConnectionId.
func AddFileToDirectory(t *testing.T, dir *directory.Directory, name string) *directory.File {
	t.Helper()

	fEvt, err := dir.NewFile(name, false)
	require.NoError(t, err)
	require.Equal(t, directory.CreateFileTriggeredType, fEvt.Type())

	f := fEvt.Payload.(directory.CreateFileTriggered).File

	err = dir.Notify(event.New(directory.CreateFileSucceeded{
		File:      f,
		Directory: dir,
	}))
	require.NoError(t, err)

	return f
}

// AddSubDirectoryToDirectory creates a new subdirectory in the provided one, then returns the new directory.
// The connection id used is FakeS3LikeConnectionId.
func AddSubDirectoryToDirectory(t *testing.T, dir *directory.Directory, name string) *directory.Directory {
	t.Helper()

	_, err := dir.NewSubDirectory(name)
	require.NoError(t, err)

	nd := NewLoadedDirectory(t, name, dir.Path())

	err = dir.Notify(event.New(directory.CreateSucceeded{
		ParentDirectory: dir,
		Directory:       nd,
	}))
	require.NoError(t, err)

	return nd
}

func AddSubNotLoadedDirectoryToDirectory(t *testing.T, dir *directory.Directory, name string) *directory.Directory {
	t.Helper()

	newEvt, err := dir.NewSubDirectory(name)
	require.NoError(t, err)
	require.Equal(t, directory.CreateTriggeredType, newEvt.Type())

	nd := newEvt.Payload.(directory.CreateTriggered).Directory

	err = dir.Notify(event.New(directory.CreateSucceeded{
		ParentDirectory: dir,
		Directory:       nd,
	}))
	require.NoError(t, err)

	return nd
}
