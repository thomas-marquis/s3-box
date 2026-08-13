package tu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/tu"
)

func TestMakeDirectory(t *testing.T) {
	fakeConnId := connection_deck.NewConnectionID()

	t.Run("should create a single root directory", func(t *testing.T) {
		// When
		res := tu.MakeDirectory(t, "", tu.AsRoot())

		// Then
		assert.Equal(t, directory.RootDirName, res.Name())
		assert.Equal(t, directory.RootPath, res.Path())
		assert.Empty(t, res.SubDirectories())
		assert.Empty(t, res.Files())
	})

	t.Run("should create a root directory with files", func(t *testing.T) {
		// When
		res := tu.MakeDirectory(t, "",
			tu.AsRoot(),
			tu.WithFile("file1.txt"), tu.WithFile("file2.txt"))

		// Then
		assert.Equal(t, directory.RootDirName, res.Name())
		assert.Equal(t, directory.RootPath, res.Path())
		assert.Empty(t, res.SubDirectories())
		assert.Len(t, res.Files(), 2)
		assert.Equal(t, "file1.txt", res.Files()[0].Name().String())
		assert.Equal(t, "file2.txt", res.Files()[1].Name().String())
	})

	t.Run("should create root directory with subdirectories", func(t *testing.T) {
		// When
		res := tu.MakeDirectory(t, "",
			tu.AsRoot(),
			tu.WithSubDirectory("sub1"),
			tu.WithSubDirectory("sub2"))

		// Then
		assert.Equal(t, directory.RootDirName, res.Name())
		assert.Equal(t, directory.RootPath, res.Path())
		assert.Len(t, res.SubDirectories(), 2)
		assert.Empty(t, res.Files())

		d1, err := res.GetSubDirectoryByName("sub1")
		assert.NoError(t, err)
		assert.Equal(t, "/sub1/", d1.Path().String())

		d2, err := res.GetSubDirectoryByName("sub2")
		assert.NoError(t, err)
		assert.Equal(t, "/sub2/", d2.Path().String())
	})

	t.Run("should create a loaded non-root directory with files", func(t *testing.T) {
		// When
		res := tu.MakeDirectory(t, "mydir",
			tu.WithRootParent(),
			tu.WithConnectionId(fakeConnId),
			tu.IsLoaded(),
			tu.WithFile("file1.txt"),
			tu.WithFile("file2.txt"),
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
		res := tu.MakeDirectory(t, "mysubdir",
			tu.WithConnectionId(fakeConnId),
			tu.WithParent("mydir",
				tu.WithRootParent(),
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

		res := tu.MakeDirectory(t, "documents",
			tu.WithConnectionId(fakeConnId),
			tu.IsLoaded(),
			tu.WithFile("report.pdf"),
			tu.WithFile("invoice.pdf"),
			tu.WithSubDirectory("code",
				tu.IsLoaded(),
				tu.WithFile("main.go"),
				tu.WithFile("test.go"),
			),
			tu.WithSubDirectory("data"),
			tu.WithParent("thomas",
				tu.WithParent("home", tu.WithRootParent()),
				tu.IsLoaded(),
				tu.WithSubDirectory("melanie"),
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

	t.Run("should return the subdirectory pointer", func(t *testing.T) {
		// When
		var res1, res2 *directory.Directory
		dir := tu.MakeDirectory(t, "mydir",
			tu.WithConnectionId(fakeConnId),
			tu.WithRootParent(),
			tu.WithSubDirectory("other",
				tu.To(&res1),
				tu.WithSubDirectory("data", tu.To(&res2))),
		)

		expected1, _ := dir.GetSubDirectoryByName("other")
		expected2, _ := expected1.GetSubDirectoryByName("data")

		// Then
		assert.NotNil(t, res1)
		assert.NotNil(t, res2)
		assert.Same(t, expected1, res1)
		assert.Same(t, expected2, res2)
	})

	t.Run("should return the file pointer", func(t *testing.T) {
		// When
		var f1, f2, f3 *directory.File
		var internalDir *directory.Directory
		dir := tu.MakeDirectory(t, "src",
			tu.WithRootParent(),
			tu.WithFile("README.md"),
			tu.WithFileTo("main.go", &f1),
			tu.WithFileTo("Makefile", &f2),
			tu.WithSubDirectory("internal",
				tu.To(&internalDir),
				tu.WithFile("utils.go"),
				tu.WithFileTo("user.go", &f3)))

		expected1, _ := dir.GetFileByName("main.go")
		expected2, _ := dir.GetFileByName("Makefile")
		expected3, _ := internalDir.GetFileByName("user.go")

		// Then
		assert.NotNil(t, f1)
		assert.NotNil(t, f2)
		assert.NotNil(t, f3)

		assert.Same(t, expected1, f1)
		assert.Same(t, expected2, f2)
		assert.Same(t, expected3, f3)
	})
}
