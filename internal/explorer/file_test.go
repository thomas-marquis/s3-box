package explorer_test

import (
	"go2s3/internal/explorer"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewRemoteFile_ShouldBuildNewFile(t *testing.T) {
	// Given
	fullPath := "path/to/file.txt"

	// When
	file := explorer.NewRemoteFile(fullPath)

	// Then
	assert.Equal(t, "file.txt", file.Name())
	assert.Equal(t, fullPath, file.Path())
	assert.Equal(t, "path/to", file.DirPath())
}
