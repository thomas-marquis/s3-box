package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestMakeDirectory(t *testing.T) {
	fakeConnId := connection_deck.NewConnectionID()

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
		// When
		res := testutil.MakeDirectory(t, "mydir",
			testutil.WithRootParent(),
			testutil.WithConnectionId(fakeConnId),
			testutil.IsLoaded(),
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
		// When
		res := testutil.MakeDirectory(t, "mysubdir",
			testutil.WithConnectionId(fakeConnId),
			testutil.WithParent("mydir",
				testutil.WithRootParent(),
			),
		)

		// Then
		assert.Equal(t, "/mydir/mysubdir/", res.Path().String())
		assert.Equal(t, fakeConnId, res.ConnectionID())

		assert.Equal(t, "/mydir/", res.Parent().Path().String())
		assert.Equal(t, fakeConnId, res.Parent().ConnectionID())

		assert.Equal(t, directory.RootPath, res.Parent().Parent().Path())
	})

	t.Run("should create a directory nested in a complex folder structure", func(t *testing.T) {
		// When

		// /
		//   home/
		//     thomas/
		//       documents/ <- res
		//         report.pdf
		//         invoice.pdf
		//         code/
		//           main.go
		//           test.go
		//         data/
		//     melanie/

		res := testutil.MakeDirectory(t, "documents",
			testutil.WithConnectionId(fakeConnId),
			testutil.IsLoaded(),
			testutil.WithFiles("report.pdf", "invoice.pdf"),
			testutil.WithSubDirectory("code",
				testutil.IsLoaded(),
				testutil.WithFiles("main.go", "test.go"),
			),
			testutil.WithSubDirectory("data"),
			testutil.WithParent("thomas",
				testutil.WithParent("home", testutil.WithRootParent()),
				testutil.IsLoaded(),
				testutil.WithSubDirectory("melanie"),
			),
		)

		// Then
		assert.Equal(t, "/home/thomas/documents/", res.Path().String())
		assert.Equal(t, fakeConnId, res.ConnectionID())
		assert.True(t, res.IsLoaded())

		assert.Len(t, res.SubDirectories(), 2)
		codeDir := res.SubDirectories()[0]
		dataDir := res.SubDirectories()[1]
		assert.Equal(t, "/home/thomas/documents/code/", codeDir.Path().String())
		assert.Equal(t, "/home/thomas/documents/data/", dataDir.Path().String())
		assert.Len(t, codeDir.Files(), 2)
		assert.Equal(t, "main.go", codeDir.Files()[0].Name().String())
		assert.Equal(t, "test.go", codeDir.Files()[1].Name().String())

		assert.Equal(t, "/home/thomas/", res.Parent().Path().String())
		assert.Equal(t, fakeConnId, res.Parent().ConnectionID())
		assert.Equal(t, "/home/", res.Parent().Parent().Path().String())
		assert.Equal(t, fakeConnId, res.Parent().Parent().ConnectionID())
		assert.Equal(t, directory.RootPath, res.Parent().Parent().Parent().Path())
	})
}
