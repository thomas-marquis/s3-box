package explorer_test

import (
	"testing"

	"github.com/thomas-marquis/s3-box/internal/explorer"

	"github.com/stretchr/testify/assert"
)

func Test_NewS3Directory_ShouldBuildDirectoryWithNonRootParent(t *testing.T) {
	// Given
	parentID := explorer.S3DirectoryID("/path/to/parent/")

	// When
	currDir, err := explorer.NewS3Directory("dir", parentID)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, explorer.S3DirectoryID("/path/to/parent/dir/"), currDir.ID)
}

func Test_NewS3Directory_ShouldBuildDirectoryWithRootParent(t *testing.T) {
	// When
	currDir, err := explorer.NewS3Directory("dir", explorer.RootDirID)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, explorer.RootDirID, currDir.ParentID)
	assert.Equal(t, explorer.S3DirectoryID("/dir/"), currDir.ID)
}

func Test_NewS3Directory_ShouldReturnErrorWhenDirectoryNameIsEmpty(t *testing.T) {
	// When
	_, err := explorer.NewS3Directory("", explorer.RootDirID)

	// Then
	assert.Error(t, err)
}

func Test_NewS3Directory_ShouldBuildNewWhenEmptyNameAndNoParentID(t *testing.T) {
	// When
	dir, err := explorer.NewS3Directory("", explorer.NilParentID)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, explorer.NilParentID, dir.ParentID)
	assert.Equal(t, "", dir.Name)
	assert.Equal(t, explorer.RootDirID, dir.ID)
}

func Test_NewS3Directory_ShouldReturnErrorWhenDirectoryNameIsSlash(t *testing.T) {
	// When
	_, err := explorer.NewS3Directory("/", explorer.RootDirID)

	// Then
	assert.Error(t, err)
}

func Test_NewS3Directory_ShouldReturnErrorWhenDirectoryNameContainsSlash(t *testing.T) {
	// When
	_, err := explorer.NewS3Directory("path/to/dir", explorer.RootDirID)

	// Then
	assert.Error(t, err)
}

func Test_AddSubDirectory_ShouldAddSubDirectory(t *testing.T) {
	// Given
	dir, err := explorer.NewS3Directory("parent", explorer.RootDirID)
	assert.NoError(t, err)

	// When
	err = dir.AddSubDirectory("subdir")
	assert.NoError(t, err)
	err = dir.AddSubDirectory("subdir2")
	assert.NoError(t, err)

	// Then
	assert.Equal(t, explorer.S3DirectoryID("/parent/subdir/"), dir.SubDirectoriesIDs[0])
	assert.Equal(t, explorer.S3DirectoryID("/parent/subdir2/"), dir.SubDirectoriesIDs[1])
}

func Test_AddSubDirectory_ShouldReturnErrorWhenSubDirectoryAlreadyExists(t *testing.T) {
	// Given
	dir, err := explorer.NewS3Directory("parent", explorer.RootDirID)
	assert.NoError(t, err)
	err = dir.AddSubDirectory("subdir")
	assert.NoError(t, err)

	// When
	err = dir.AddSubDirectory("subdir")

	// Then
	assert.Error(t, err)
	assert.Equal(t, "sub directory /parent/subdir/ already exists in S3 directory /parent/", err.Error())
}

func Test_CreateEmptyS3Directory_ShouldReturnEmptyDirectory(t *testing.T) {
	// Given
	dir, _ := explorer.NewS3Directory("dir", explorer.RootDirID)

	// When
	newDir, err := dir.CreateEmptySubDirectory("subdir")

	// Then
	assert.NoError(t, err)
	assert.Equal(t, "subdir", newDir.Name, "Name should be 'subdir'")
	assert.Equal(t, explorer.S3DirectoryID("/dir/subdir/"), newDir.ID, "ID should be '/dir/subdir/'")
	assert.Len(t, dir.SubDirectoriesIDs, 1, "SubDirectoriesIDs should contain one element")
}

func Test_CreateEmptyS3Directory_ShouldReturnErrorWhenSubDirectoryAlreadyExists(t *testing.T) {
	// Given
	dir, _ := explorer.NewS3Directory("dir", explorer.RootDirID)
	_, _ = dir.CreateEmptySubDirectory("subdir")

	// When
	newDir, err := dir.CreateEmptySubDirectory("subdir")

	// Then
	assert.Error(t, err)
	assert.Equal(t, "sub directory /dir/subdir/ already exists in S3 directory /dir/", err.Error())
	assert.Nil(t, newDir)
}

func Test_RemoveSubDirectory_ShouldRemoveSubDirectoryWhenExists(t *testing.T) {
	// Given
	dir, _ := explorer.NewS3Directory("dir", explorer.RootDirID)
	subDir, _ := dir.CreateEmptySubDirectory("subdir")

	// When
	err := dir.RemoveSubDirectory(subDir.ID)

	// Then
	assert.NoError(t, err)
	assert.Len(t, dir.SubDirectoriesIDs, 0, "SubDirectoriesIDs should be empty")
}

func Test_RemoveSubDirecotry_ShoudlReturnErrorWhenSubDirNotExists(t *testing.T) {
	// Given
	dir, _ := explorer.NewS3Directory("dir", explorer.RootDirID)
	dir.CreateEmptySubDirectory("subdir")

	// When
	err := dir.RemoveSubDirectory(explorer.S3DirectoryID("/bin/"))

	// Then
	assert.Error(t, err)
	assert.Equal(t, explorer.ErrObjectNotFoundInDirectory, err)
}
