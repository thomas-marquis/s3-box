package directory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func Test_New_ShouldBuildDirectoryWithNonRootParent(t *testing.T) {
	// Given
	parentPath := directory.NewPath("/path/to/parent/")

	// When
	currDir, err := directory.New(connection_deck.NewConnectionID(), "dir", parentPath)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, directory.NewPath("/path/to/parent/dir/"), currDir.Path())
}

func Test_New_ShouldBuildDirectoryWithRootParent(t *testing.T) {
	// When
	currDir, err := directory.New(connection_deck.NewConnectionID(), "dir", directory.RootPath)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, directory.RootPath, currDir.ParentPath())
	assert.Equal(t, directory.NewPath("/dir/"), currDir.Path())
}

func Test_New_ShouldReturnErrorWhenDirectoryNameIsEmpty(t *testing.T) {
	// When
	_, err := directory.New(connection_deck.NewConnectionID(), "", directory.RootPath)

	// Then
	assert.Error(t, err)
}

func Test_New_ShouldBuildNewWhenEmptyNameAndNoParentID(t *testing.T) {
	// When
	dir, err := directory.New(connection_deck.NewConnectionID(), "", directory.NilParentPath)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, directory.NilParentPath, dir.ParentPath())
	assert.Equal(t, "", dir.Name)
	assert.Equal(t, directory.RootPath, dir.Path())
}

func Test_New_ShouldReturnErrorWhenDirectoryNameIsSlash(t *testing.T) {
	// When
	_, err := directory.New(connection_deck.NewConnectionID(), "/", directory.RootPath)

	// Then
	assert.Error(t, err)
}

func Test_New_ShouldReturnErrorWhenDirectoryNameContainsSlash(t *testing.T) {
	// When
	_, err := directory.New(connection_deck.NewConnectionID(), "path/to/dir", directory.RootPath)

	// Then
	assert.Error(t, err)
}

func Test_AddSubDirectory_ShouldAddSubDirectory(t *testing.T) {
	// Given
	dir, err := directory.New(connection_deck.NewConnectionID(), "parent", directory.RootPath)
	assert.NoError(t, err)

	// When
	res1, err := dir.NewSubDirectory("subdir")
	assert.NoError(t, err)
	res2, err := dir.NewSubDirectory("subdir2")
	assert.NoError(t, err)

	// Then
	assert.Len(t, dir.SubDirectories(), 2, "SubDirectories should contain two elements")
	assert.Equal(t, directory.NewPath("/parent/subdir/"), dir.SubDirectories()[0])
	assert.Equal(t, directory.NewPath("/parent/subdir2/"), dir.SubDirectories()[1])
	assert.Equal(t, res1.Directory(), dir.SubDirectories()[0], "First subdirectory should match the first added")
	assert.Equal(t, res2.Directory(), dir.SubDirectories()[1], "Second subdirectory should match the second added")
}

func Test_NewSubDirectory_ShouldReturnEmptyDirectory(t *testing.T) {
	// Given
	dir, _ := directory.New(connection_deck.NewConnectionID(), "dir", directory.RootPath)

	// When
	evt, err := dir.NewSubDirectory("subdir")

	// Then
	assert.NoError(t, err)

	assert.Equal(t, directory.CreatedEventName, evt.Name())
	assert.Equal(t, dir, evt.Directory())

	newDir := evt.Directory()
	assert.Equal(t, "subdir", newDir.Name(), "Name should be 'subdir'")
	assert.Equal(t, directory.NewPath("/dir/subdir/"), newDir.Path(), "Path should be '/dir/subdir/'")
	assert.Len(t, dir.SubDirectories(), 1, "SubDirectories should contain one element")
}

func Test_NewSubDirectory_ShouldReturnErrorWhenSubDirectoryAlreadyExists(t *testing.T) {
	// Given
	dir, _ := directory.New(connection_deck.NewConnectionID(), "dir", directory.RootPath)
	_, _ = dir.NewSubDirectory("subdir")

	// When
	_, err := dir.NewSubDirectory("subdir")

	// Then
	assert.Error(t, err)
	assert.Equal(t, "sub directory /dir/subdir/ already exists in S3 directory /dir/", err.Error())
}

func Test_RemoveSubDirectory_ShouldRemoveSubDirectoryWhenExists(t *testing.T) {
	// Given
	dir, _ := directory.New(connection_deck.NewConnectionID(), "dir", directory.RootPath)
	evt, _ := dir.NewSubDirectory("subdir")
	subDir := evt.Directory()

	// When
	evt, err := dir.RemoveSubDirectory(subDir.Name())

	// Then
	assert.NoError(t, err)
	assert.Len(t, dir.SubDirectories(), 0, "SubDirectories should be empty")
	assert.Equal(t, directory.DeletedEventName, evt.Name())
	assert.Equal(t, dir.Path(), evt.Directory().ParentPath())
}

func Test_RemoveSubDirectory_ShouldReturnErrorWhenSubDirNotExists(t *testing.T) {
	// Given
	dir, _ := directory.New(connection_deck.NewConnectionID(), "dir", directory.RootPath)
	dir.NewSubDirectory("subdir")

	// When
	_, err := dir.RemoveSubDirectory("bin")

	// Then
	assert.Error(t, err)
	assert.Equal(t, directory.ErrNotFound, err)
}
