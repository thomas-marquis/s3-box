package directory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connections"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func Test_New_ShouldBuildDirectoryWithNonRootParent(t *testing.T) {
	// Given
	parentPath := directory.NewPath("/path/to/parent/")
	updates := make(chan directory.Update, 10)

	// When
	currDir, err := directory.New(connections.NewConnectionID(), "dir", parentPath, updates)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, directory.NewPath("/path/to/parent/dir/"), currDir.Path())
}

func Test_New_ShouldBuildDirectoryWithRootParent(t *testing.T) {
	// When
	updates := make(chan directory.Update, 10)
	currDir, err := directory.New(connections.NewConnectionID(), "dir", directory.RootPath, updates)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, directory.RootPath, currDir.ParentPath())
	assert.Equal(t, directory.NewPath("/dir/"), currDir.Path())
}

func Test_New_ShouldReturnErrorWhenDirectoryNameIsEmpty(t *testing.T) {
	// When
	updates := make(chan directory.Update, 10)
	_, err := directory.New(connections.NewConnectionID(), "", directory.RootPath, updates)

	// Then
	assert.Error(t, err)
}

func Test_New_ShouldBuildNewWhenEmptyNameAndNoParentID(t *testing.T) {
	// When
	updates := make(chan directory.Update, 10)
	dir, err := directory.New(connections.NewConnectionID(), "", directory.NilParentPath, updates)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, directory.NilParentPath, dir.ParentPath())
	assert.Equal(t, "", dir.Name)
	assert.Equal(t, directory.RootPath, dir.Path())
}

func Test_New_ShouldReturnErrorWhenDirectoryNameIsSlash(t *testing.T) {
	// When
	updates := make(chan directory.Update, 10)
	_, err := directory.New(connections.NewConnectionID(), "/", directory.RootPath, updates)

	// Then
	assert.Error(t, err)
}

func Test_New_ShouldReturnErrorWhenDirectoryNameContainsSlash(t *testing.T) {
	// When
	updates := make(chan directory.Update, 10)
	_, err := directory.New(connections.NewConnectionID(), "path/to/dir", directory.RootPath, updates)

	// Then
	assert.Error(t, err)
}

func Test_AddSubDirectory_ShouldAddSubDirectory(t *testing.T) {
	// Given
	updates := make(chan directory.Update, 10)
	dir, err := directory.New(connections.NewConnectionID(), "parent", directory.RootPath, updates)
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
	assert.Equal(t, res1, dir.SubDirectories()[0], "First subdirectory should match the first added")
	assert.Equal(t, res2, dir.SubDirectories()[1], "Second subdirectory should match the second added")
}

func Test_NewSubDirectory_ShouldReturnEmptyDirectory(t *testing.T) {
	// Given
	updates := make(chan directory.Update, 10)
	dir, _ := directory.New(connections.NewConnectionID(), "dir", directory.RootPath, updates)

	// When
	newDir, err := dir.NewSubDirectory("subdir")

	// Then
	assert.NoError(t, err)
	assert.Equal(t, "subdir", newDir.Name(), "Name should be 'subdir'")
	assert.Equal(t, directory.NewPath("/dir/subdir/"), newDir.Path(), "Path should be '/dir/subdir/'")
	assert.Len(t, dir.SubDirectories(), 1, "SubDirectories should contain one element")
}

func Test_NewSubDirectory_ShouldReturnErrorWhenSubDirectoryAlreadyExists(t *testing.T) {
	// Given
	updates := make(chan directory.Update, 10)
	dir, _ := directory.New(connections.NewConnectionID(), "dir", directory.RootPath, updates)
	_, _ = dir.NewSubDirectory("subdir")

	// When
	newDir, err := dir.NewSubDirectory("subdir")

	// Then
	assert.Error(t, err)
	assert.Equal(t, "sub directory /dir/subdir/ already exists in S3 directory /dir/", err.Error())
	assert.Nil(t, newDir)
}

func Test_RemoveSubDirectory_ShouldRemoveSubDirectoryWhenExists(t *testing.T) {
	// Given
	updates := make(chan directory.Update, 10)
	dir, _ := directory.New(connections.NewConnectionID(), "dir", directory.RootPath, updates)
	subDir, _ := dir.NewSubDirectory("subdir")

	// When
	err := dir.RemoveSubDirectory(subDir.Name())

	// Then
	assert.NoError(t, err)
	assert.Len(t, dir.SubDirectories(), 0, "SubDirectories should be empty")
}

func Test_RemoveSubDirecotry_ShoudlReturnErrorWhenSubDirNotExists(t *testing.T) {
	// Given
	updates := make(chan directory.Update, 10)
	dir, _ := directory.New(connections.NewConnectionID(), "dir", directory.RootPath, updates)
	dir.NewSubDirectory("subdir")

	// When
	err := dir.RemoveSubDirectory("bin")

	// Then
	assert.Error(t, err)
	assert.Equal(t, directory.ErrNotFound, err)
}
