package explorer_test

import (
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewRemoteFile_ShouldBuildNewFile(t *testing.T) {
	// Given
	fullPath := "path/to/file.txt"
	rootDir := explorer.NewDirectory("root", nil)

	// When
	file := explorer.NewS3File(fullPath, rootDir)

	// Then
	assert.Equal(t, "file.txt", file.Name())
	assert.Equal(t, fullPath, file.Path())
	assert.Equal(t, "path/to", file.DirPath())
}
