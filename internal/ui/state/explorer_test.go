package state_test

import (
	"testing"

	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"github.com/thomas-marquis/s3-box/internal/ui/node"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
)

func TestExplorerState_InitFileTree(t *testing.T) {
	fyne_test.NewTempApp(t)

	t.Run("should create the root node from the provided root directory", func(t *testing.T) {
		// Given
		s := state.New()
		rootDir := testutil.MakeDirectory(t, "", testutil.AsRoot())

		// When
		err := s.Explorer().InitFileTree(rootDir, "myBucket")

		// Then
		assert.NoError(t, err)

		childIds, values, err := s.Explorer().FileTree().Get()
		require.NoError(t, err)

		assert.Equal(t, map[string]node.Node{
			"/": node.NewDirectoryNode(rootDir, node.WithDisplayName("Bucket: myBucket")),
		}, values)
		assert.Equal(t, map[string][]string{
			"": {"/"},
		}, childIds)
	})
}

func TestExplorerState_AppendFile(t *testing.T) {
	fyne_test.NewTempApp(t)

	t.Run("should append a file to an empty tree", func(t *testing.T) {
		// Given
		s := state.New()
		var f *directory.File
		rootDir := testutil.MakeDirectory(t, "",
			testutil.AsRoot(),
			testutil.WithFiles("file.txt"),
			testutil.FileTo("file.txt", &f))
		require.NoError(t, s.Explorer().InitFileTree(rootDir, "myBucket"))

		// When
		err := s.Explorer().AppendFile(f)

		// Then
		assert.NoError(t, err)

		childIds, values, err := s.Explorer().FileTree().Get()
		require.NoError(t, err)

		assert.Equal(t, map[string]node.Node{
			"/":         node.NewDirectoryNode(rootDir, node.WithDisplayName("Bucket: myBucket")),
			"/file.txt": node.NewFileNode(f),
		}, values)
		assert.Equal(t, map[string][]string{
			"":  {"/"},
			"/": {"/file.txt"},
		}, childIds)
	})

	t.Run("should append a file from a non empty tree", func(t *testing.T) {
		// Given
		s := state.New()

		var mainFile, readmeFile, userFile *directory.File
		var srcDir *directory.Directory

		rootDir := testutil.MakeDirectory(t, "",
			testutil.AsRoot(),
			testutil.WithFiles("main.go", "README.md"),
			testutil.FileTo("main.go", &mainFile),
			testutil.FileTo("README.md", &readmeFile),
			testutil.WithSubDirectory("src",
				testutil.To(&srcDir),
				testutil.WithFiles("user.go"),
				testutil.FileTo("user.go", &userFile)))
		require.NoError(t, s.Explorer().InitFileTree(rootDir, "myBucket"))

		require.NoError(t, s.Explorer().AppendFile(mainFile))
		require.NoError(t, s.Explorer().AppendFile(readmeFile))
		require.NoError(t, s.Explorer().PrependDirectory(srcDir))
		require.NoError(t, s.Explorer().AppendFile(userFile))

		// When
		utilsFile, _ := directory.NewFile("utils.go", srcDir)
		err := s.Explorer().AppendFile(utilsFile)

		// Then
		assert.NoError(t, err)

		childIds, values, err := s.Explorer().FileTree().Get()
		require.NoError(t, err)

		assert.Equal(t, map[string]node.Node{
			"/":             node.NewDirectoryNode(rootDir, node.WithDisplayName("Bucket: myBucket")),
			"/main.go":      node.NewFileNode(mainFile),
			"/README.md":    node.NewFileNode(readmeFile),
			"/src/":         node.NewDirectoryNode(srcDir),
			"/src/user.go":  node.NewFileNode(userFile),
			"/src/utils.go": node.NewFileNode(utilsFile),
		}, values)
		assert.Equal(t, map[string][]string{
			"":      {"/"},
			"/":     {"/src/", "/main.go", "/README.md"},
			"/src/": {"/src/user.go", "/src/utils.go"},
		}, childIds)
	})
}

func TestExplorerState_PrependDirectory(t *testing.T) {
	fyne_test.NewTempApp(t)

	t.Run("should prepend a directory to an empty tree", func(t *testing.T) {
		// Given
		s := state.New()
		var dataDir *directory.Directory
		rootDir := testutil.MakeDirectory(t, "",
			testutil.AsRoot(),
			testutil.IsLoaded(),
			testutil.WithSubDirectory("data",
				testutil.To(&dataDir)))
		require.NoError(t, s.Explorer().InitFileTree(rootDir, "myBucket"))

		// When
		err := s.Explorer().PrependDirectory(dataDir)

		// Then
		assert.NoError(t, err)

		childIds, values, err := s.Explorer().FileTree().Get()
		require.NoError(t, err)

		assert.Equal(t, map[string]node.Node{
			"/":      node.NewDirectoryNode(rootDir, node.WithDisplayName("Bucket: myBucket")),
			"/data/": node.NewDirectoryNode(dataDir),
		}, values)
		assert.Equal(t, map[string][]string{
			"":  {"/"},
			"/": {"/data/"},
		}, childIds)
	})

	t.Run("should prepend a directory from a non empty tree", func(t *testing.T) {
		// Given
		s := state.New()

		var mainFile, readmeFile, userFile *directory.File
		var srcDir, infraDir *directory.Directory

		rootDir := testutil.MakeDirectory(t, "",
			testutil.AsRoot(),
			testutil.WithFiles("main.go", "README.md"),
			testutil.FileTo("main.go", &mainFile),
			testutil.FileTo("README.md", &readmeFile),
			testutil.WithSubDirectory("src",
				testutil.To(&srcDir),
				testutil.WithFiles("user.go"),
				testutil.FileTo("user.go", &userFile),
				testutil.WithSubDirectory("infra",
					testutil.To(&infraDir))))
		require.NoError(t, s.Explorer().InitFileTree(rootDir, "myBucket"))

		require.NoError(t, s.Explorer().AppendFile(mainFile))
		require.NoError(t, s.Explorer().AppendFile(readmeFile))
		require.NoError(t, s.Explorer().PrependDirectory(srcDir))
		require.NoError(t, s.Explorer().AppendFile(userFile))

		// When
		err := s.Explorer().PrependDirectory(infraDir)

		// Then
		assert.NoError(t, err)

		childIds, values, err := s.Explorer().FileTree().Get()
		require.NoError(t, err)

		assert.Equal(t, map[string]node.Node{
			"/":            node.NewDirectoryNode(rootDir, node.WithDisplayName("Bucket: myBucket")),
			"/main.go":     node.NewFileNode(mainFile),
			"/README.md":   node.NewFileNode(readmeFile),
			"/src/":        node.NewDirectoryNode(srcDir),
			"/src/user.go": node.NewFileNode(userFile),
			"/src/infra/":  node.NewDirectoryNode(infraDir),
		}, values)
		assert.Equal(t, map[string][]string{
			"":      {"/"},
			"/":     {"/src/", "/main.go", "/README.md"},
			"/src/": {"/src/infra/", "/src/user.go"},
		}, childIds)
	})

	t.Run("should return an error if the parent is not in the tree", func(t *testing.T) {
		// Given
		s := state.New()
		var dataDir, csvDir *directory.Directory
		rootDir := testutil.MakeDirectory(t, "",
			testutil.AsRoot(),
			testutil.WithSubDirectory("data",
				testutil.To(&dataDir),
				testutil.WithSubDirectory("csv",
					testutil.To(&csvDir))))
		require.NoError(t, s.Explorer().InitFileTree(rootDir, "myBucket"))

		require.NoError(t, s.Explorer().InitFileTree(rootDir, "myBucket"))

		// When
		err := s.Explorer().PrependDirectory(csvDir)

		// Then
		var sErr state.Error
		assert.ErrorAs(t, err, &sErr)
		assert.ErrorContains(t, err, "failed prepending the directory '/data/csv/' to file tree because its parents has not been found")
	})
}
