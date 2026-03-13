package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

// FakeRootDirectory creates a new root directory with FakeS3LikeConnectionId
func FakeRootDirectory(t *testing.T) *directory.Directory {
	t.Helper()

	dir, err := directory.New(FakeS3LikeConnectionId, directory.RootDirName, directory.NilParentPath)
	require.NoError(t, err)

	return dir
}

// NewDirectory creates a new directory with FakeS3LikeConnectionId
func NewDirectory(t *testing.T, name string, parent directory.Path) *directory.Directory {
	t.Helper()

	dir, err := directory.New(FakeS3LikeConnectionId, name, parent)
	require.NoError(t, err)

	_, err = dir.Load()
	require.NoError(t, err)

	err = dir.Notify(directory.NewLoadSuccessEvent(dir, nil, nil))
	require.NoError(t, err)

	return dir
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

	nd := NewDirectory(t, name, dir.Path())

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
