package directory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func Test_NewFile_ShouldBuildNewFile(t *testing.T) {
	// Given
	fileName := "file.txt"
	parentDir, _ := directory.New(connection_deck.NewConnectionID(), "path", directory.RootPath)

	// When
	file, err := directory.NewFile(fileName, parentDir)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, fileName, file.Name)
	assert.Equal(t, parentDir.Path(), file.DirectoryPath())
	assert.Equal(t, directory.FileName("file.txt"), file.Name())
}

func Test_NewFile_ShouldBuildNewFileWithNonRootParent(t *testing.T) {
	// Given
	fileName := "file.txt"
	parentDir, _ := directory.New(connection_deck.NewConnectionID(), "a_directory", directory.NewPath("/path/to/parent/"))

	// When
	file, err := directory.NewFile(fileName, parentDir)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, fileName, file.Name)
	assert.Equal(t, parentDir.Path(), file.DirectoryPath())
	assert.Equal(t, directory.FileName("file.txt"), file.Name())
}

func Test_NewFile_ShouldReturnErrorWhenNameIsEmpty(t *testing.T) {
	// Given
	fileName := ""
	parentDir, _ := directory.New(connection_deck.NewConnectionID(), "path", directory.RootPath)

	// When
	file, err := directory.NewFile(fileName, parentDir)

	// Then
	assert.Error(t, err)
	assert.Equal(t, "file name is empty", err.Error())
	assert.Nil(t, file)
}

func Test_NewFile_ShouldReturnErrorWhenNameIsNotValid(t *testing.T) {
	// Given
	fileName := "/"
	parentDir, _ := directory.New(connection_deck.NewConnectionID(), "path", directory.RootPath)

	// When
	file, err := directory.NewFile(fileName, parentDir)

	// Then
	assert.Error(t, err)
	assert.Equal(t, "file name is not valid", err.Error())
	assert.Nil(t, file)
}
