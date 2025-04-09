package explorer_test

import (
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Path_ShouldReturnFullPathWhenParentDirIsNotNil(t *testing.T) {
	// Given
	parentDir := explorer.NewS3Directory("parent", explorer.RootDir)
	currDir := explorer.NewS3Directory("dir", parentDir)

	// When
	path := currDir.Path()

	// Then
	assert.Equal(t, "/parent/dir", path)
}

func Test_NewS3Directory_ShouldSetRootDirAsParentByDefault(t *testing.T) {
	// Given
	parentDir := explorer.NewS3Directory("parent", nil)

	// When
	currDir := explorer.NewS3Directory("dir", parentDir)

	// Then
	assert.Equal(t, explorer.RootDir, currDir.Parrent.Parrent)
	assert.Equal(t, "/parent/dir", currDir.Path())
}
