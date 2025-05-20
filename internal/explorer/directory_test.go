package explorer_test

import (
	"testing"

	"github.com/thomas-marquis/s3-box/internal/explorer"

	"github.com/stretchr/testify/assert"
)

func Test_S3DirectoryID_ToName_ShouldReturnNameOfDirectory(t *testing.T) {
	testCases := []struct {
		id           explorer.S3DirectoryID
		expectedName string
	}{
		{explorer.RootDirID, ""},
		{explorer.S3DirectoryID("/"), ""},
		{explorer.S3DirectoryID(""), ""},
		{explorer.S3DirectoryID("/path"), "path"},
		{explorer.S3DirectoryID("/path/"), "path"},
		{explorer.S3DirectoryID("/path/to/dir"), "dir"},
		{explorer.S3DirectoryID("/path/to/dir/"), "dir"},
		{explorer.S3DirectoryID("path/to/dir/subdir"), "subdir"},
		{explorer.S3DirectoryID("path/to/dir/subdir/"), "subdir"},
		{explorer.S3DirectoryID("/path/to/dir/subdir/subsubdir"), "subsubdir"},
		{explorer.S3DirectoryID("/path/to/dir/subdir/subsubdir/"), "subsubdir"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id.String(), func(t *testing.T) {
			// When
			name := testCase.id.ToName()

			// Then
			assert.Equal(t, testCase.expectedName, name)
		})
	}
}

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
	assert.Equal(t, explorer.S3DirectoryID("parent/subdir"), dir.SubDirectoriesIDs[0])
	assert.Equal(t, explorer.S3DirectoryID("parent/subdir2"), dir.SubDirectoriesIDs[1])
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
	assert.Equal(t, "sub directory parent/subdir already exists in S3 directory parent", err.Error())
}
