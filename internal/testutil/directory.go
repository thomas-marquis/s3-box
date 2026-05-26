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
	return FakeLoadedRootDirectoryWithConn(t, FakeS3LikeConnectionId)
}

// FakeLoadedRootDirectoryWithConn creates a new root directory (loaded) with connection ID
func FakeLoadedRootDirectoryWithConn(t *testing.T, connId connection_deck.ConnectionID) *directory.Directory {
	t.Helper()

	dir, err := directory.NewRoot(connId)
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

type subDirectoryBuilderConfig struct {
	files          []string
	loaded         bool
	subDirectories []*directory.Directory
}

type directoryBuilderConfig struct {
	loaded         bool
	subDirectories []*directory.Directory
	files          []string
	connectionId   connection_deck.ConnectionID
	parent         *directory.Directory
	hasRootParent  bool
}

type DirectoryBuilderOption func(*directoryBuilderConfig)
type SubDirectoryBuilderOption func(*subDirectoryBuilderConfig)

func WithLoaded(loaded bool) DirectoryBuilderOption {
	return func(cfg *directoryBuilderConfig) {
		cfg.loaded = loaded
	}
}

func WithConnectionId(connId connection_deck.ConnectionID) DirectoryBuilderOption {
	return func(cfg *directoryBuilderConfig) {
		cfg.connectionId = connId
	}
}

func WithFiles(fileNames ...string) DirectoryBuilderOption {
	return func(cfg *directoryBuilderConfig) {
		cfg.files = fileNames
	}
}

func WithSubDirectories(dirs ...*directory.Directory) DirectoryBuilderOption {
	return func(cfg *directoryBuilderConfig) {
		cfg.subDirectories = dirs
	}
}

func WithSubDirectory(name string) SubDirectoryBuilderOption {
	return func(cfg *subDirectoryBuilderConfig) {

	}
}

func WithParent(parent *directory.Directory) DirectoryBuilderOption {
	return func(cfg *directoryBuilderConfig) {
		cfg.parent = parent
	}
}

func WithRootParent() DirectoryBuilderOption {
	return func(cfg *directoryBuilderConfig) {
		cfg.hasRootParent = true
	}
}

func MakeDirectory(t *testing.T, name string, opts ...DirectoryBuilderOption) *directory.Directory {
	t.Helper()

	cfg := directoryBuilderConfig{
		connectionId: FakeAwsConnectionId,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var dir *directory.Directory
	var err error
	if cfg.hasRootParent {
		root, errRoot := directory.NewRoot(cfg.connectionId)
		require.NoError(t, errRoot)
		_, errRoot = root.Load()
		require.NoError(t, errRoot)
		errRoot = root.Notify(event.New(directory.LoadSucceeded{
			Directory: root,
		}))
		require.NoError(t, errRoot)

		dir, err = directory.New(cfg.connectionId, name, root)
	} else if cfg.parent != nil {
		dir, err = directory.New(cfg.connectionId, name, cfg.parent)
	} else {
		dir, err = directory.NewRoot(cfg.connectionId)
	}
	require.NoError(t, err)

	if len(cfg.files) > 0 {
		for _, fileName := range cfg.files {
			fEvt, err := dir.NewFile(fileName, false)
			require.NoError(t, err)

			f := fEvt.Payload.(directory.CreateFileTriggered).File

			err = dir.Notify(event.New(directory.CreateFileSucceeded{
				File:      f,
				Directory: dir,
			}))
			require.NoError(t, err)
		}
	}

	if len(cfg.subDirectories) > 0 {
		for _, subDir := range cfg.subDirectories {
			err = dir.Notify(event.New(directory.CreateSucceeded{
				ParentDirectory: dir,
				Directory:       subDir,
			}))
			require.NoError(t, err)
		}
	}

	if cfg.loaded {
		_, err = dir.Load()
		require.NoError(t, err)
		err = dir.Notify(event.New(directory.LoadSucceeded{
			Directory: dir,
		}))
		require.NoError(t, err)
	}

	return dir
}
