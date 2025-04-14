package explorer_test

import (
	"testing"

	"github.com/thomas-marquis/s3-box/internal/explorer"

	"github.com/stretchr/testify/assert"
)

func Test_S3FileID_ToName_ShouldReturnNameOfFile(t *testing.T) {
	testCases := []struct {
		id explorer.S3FileID
		expectedName string
	}{
		{explorer.S3FileID("file.txt"), "file.txt"},
		{explorer.S3FileID("/file.txt"), "file.txt"},
		{explorer.S3FileID("/path/to/file.txt"), "file.txt"},
		{explorer.S3FileID("path/to/file.txt"), "file.txt"},
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

func Test_NewS3File_ShouldBuildNewFile(t *testing.T) {
	// Given
	fileName := "file.txt"
	parentDir, _ := explorer.NewS3Directory("path", explorer.RootDirID)

	// When
	file, err := explorer.NewS3File(fileName, parentDir)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, fileName, file.Name)
	assert.Equal(t, parentDir.ID, file.DirectoryID)
	assert.Equal(t, explorer.S3FileID("/path/file.txt"), file.ID)
}

func Test_NewS3File_ShouldBuildNewFileWithNonRootParent(t *testing.T) {
	// Given
	fileName := "file.txt"
	parentDir, _ := explorer.NewS3Directory("a_directory", explorer.S3DirectoryID("/path/to/parent"))

	// When
	file, err := explorer.NewS3File(fileName, parentDir)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, fileName, file.Name)
	assert.Equal(t, parentDir.ID, file.DirectoryID)
	assert.Equal(t, explorer.S3FileID("/path/to/parent/a_directory/file.txt"), file.ID)
}

func Test_NewS3File_ShouldReturnErrorWhenNameIsEmpty(t *testing.T) {
	// Given
	fileName := ""
	parentDir, _ := explorer.NewS3Directory("path", explorer.RootDirID)

	// When
	file, err := explorer.NewS3File(fileName, parentDir)

	// Then
	assert.Error(t, err)
	assert.Equal(t, "file name is empty", err.Error())
	assert.Nil(t, file)
}

func Test_NewS3File_ShouldReturnErrorWhenNameIsNotValid(t *testing.T) {
	// Given
	fileName := "/"
	parentDir, _ := explorer.NewS3Directory("path", explorer.RootDirID)

	// When
	file, err := explorer.NewS3File(fileName, parentDir)

	// Then
	assert.Error(t, err)
	assert.Equal(t, "file name is not valid", err.Error())
	assert.Nil(t, file)
}
