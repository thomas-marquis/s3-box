package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestMakeDirectory(t *testing.T) {
	t.Run("should create a single root directory", func(t *testing.T) {
		// When
		res := testutil.MakeDirectory(t, "", testutil.AsRoot())

		// Then
		assert.Equal(t, directory.RootDirName, res.Name())
		assert.Equal(t, directory.RootPath, res.Path())
		assert.Empty(t, res.SubDirectories())
		assert.Empty(t, res.Files())
	})

	t.Run("should create a loaded non-root directory with files", func(t *testing.T) {
		// Given
		fakeConnId := connection_deck.NewConnectionID()

		// When
		res := testutil.MakeDirectory(t, "mydir",
			testutil.WithRootParent(),
			testutil.WithConnectionId(fakeConnId),
			testutil.WithLoaded(true),
			testutil.WithFiles("file1.txt", "file2.txt"),
		)

		// Then
		assert.Equal(t, "mydir", res.Name())
		assert.Equal(t, "/mydir/", res.Path().String())
		assert.Empty(t, res.SubDirectories())
		assert.Len(t, res.Files(), 2)
		assert.Equal(t, "file1.txt", res.Files()[0].Name().String())
		assert.Equal(t, "file2.txt", res.Files()[1].Name().String())
		assert.True(t, res.IsLoaded())
	})

	t.Run("should create a directory with a non-root parent", func(t *testing.T) {
		// Given
		fakeConnId := connection_deck.NewConnectionID()

		// When
		res := testutil.MakeDirectory(t, "mysubdir",
			testutil.WithParent("mydir",
				testutil.AsRoot(),
			),
		)

		// Then
		assert.Equal(t, "/mydir/mysubdir/", res.Path().String())
		assert.Equal(t, fakeConnId, res.ConnectionID())

		assert.Equal(t, "/mydir/", res.Parent().Path().String())
		assert.Equal(t, fakeConnId, res.Parent().ConnectionID())

		assert.Equal(t, directory.RootPath, res.Parent().Parent().Path())
	})
}
