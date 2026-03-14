package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
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

	err = dir.Notify(directory.NewLoadSuccessEvent(dir, nil, nil))
	require.NoError(t, err)

	return dir
}

func newLoadedDirectory(t *testing.T, name string, parent *directory.Directory) *directory.Directory {
	dir, err := directory.New(FakeS3LikeConnectionId, name, parent)
	require.NoError(t, err)

	_, err = dir.Load()
	require.NoError(t, err)

	err = dir.Notify(directory.NewLoadSuccessEvent(dir, nil, nil))
	require.NoError(t, err)

	return dir
}

func newNotLoadedDirectory(t *testing.T, name string, parent *directory.Directory) *directory.Directory {
	dir, err := directory.New(FakeS3LikeConnectionId, name, parent)
	require.NoError(t, err)

	return dir
}

// NewLoadedDirectory creates a new loaded directory with FakeS3LikeConnectionId
func NewLoadedDirectory(t *testing.T, name string, parentPath directory.Path) *directory.Directory {
	t.Helper()

	parent := FakeNotLoadedRootDirectory(t)
	if parentPath != directory.RootPath {
		for _, name := range parentPath.Split() {
			parent = newLoadedDirectory(t, name, parent)
		}
	}

	return newLoadedDirectory(t, name, parent)
}

// NewNotLoadedDirectory creates a new unloaded directory with FakeS3LikeConnectionId, but with loaded parents chain
func NewNotLoadedDirectory(t *testing.T, name string, parentPath directory.Path) *directory.Directory {
	t.Helper()

	parent := FakeNotLoadedRootDirectory(t)
	if parentPath != directory.RootPath {
		for _, name := range parentPath.Split() {
			parent = newLoadedDirectory(t, name, parent) // a not loaded dir with an unloaded parent doesn't make any sense
		}
	}

	return newNotLoadedDirectory(t, name, parent)
}

// AddFileToDirectory creates a new file in the provided directory, then returns the new file.
// The connection id used is FakeS3LikeConnectionId.
func AddFileToDirectory(t *testing.T, dir *directory.Directory, name string) *directory.File {
	t.Helper()

	fEvt, err := dir.NewFile(name, false)
	require.NoError(t, err)

	f := fEvt.File()

	err = dir.Notify(directory.NewFileCreatedSuccessEvent(dir, f))
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

	err = dir.Notify(directory.NewCreatedSuccessEvent(dir, nd))
	require.NoError(t, err)

	return nd
}

func AddSubNotLoadedDirectoryToDirectory(t *testing.T, dir *directory.Directory, name string) *directory.Directory {
	t.Helper()

	newEvt, err := dir.NewSubDirectory(name)
	require.NoError(t, err)

	nd := newEvt.Directory()

	err = dir.Notify(directory.NewCreatedSuccessEvent(dir, nd))
	require.NoError(t, err)

	return nd
}
