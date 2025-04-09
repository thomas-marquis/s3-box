package explorer_test

import (
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewS3File_ShouldBuildNewFile(t *testing.T) {
	// Given
	fullPath := "path/to/file.txt"

	// When
	file := explorer.NewS3File(fullPath)

	// Then
	assert.Equal(t, "file.txt", file.Name())
	assert.Equal(t, fullPath, file.Path())
	assert.Equal(t, "path/to", file.DirPath())
}
