package directory_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestPath_DirectoryName(t *testing.T) {
	for _, tc := range []struct {
		In       string
		Expected string
	}{
		{"/home/user/data/", "data"},
		{"/data/", "data"},
		{"/home/user/data", "data"},
		{"home/user/data", "data"},
		{directory.RootPath.String(), directory.RootDirName},
	} {
		t.Run(fmt.Sprintf("should return directory name for %s", tc.In), func(t *testing.T) {
			// Given
			p := directory.NewPath(tc.In)

			// When
			res := p.DirectoryName()

			// Then
			assert.Equal(t, tc.Expected, res)
		})
	}
}

func TestPath_ParentPath(t *testing.T) {
	for _, tc := range []struct {
		Path     string
		Expected string
	}{
		{"/home/user/data/", "/home/user/"},
		{"/data/", "/"},
		{"/data/", directory.RootPath.String()},
		{"/home/user/data", "/home/user/"},
		{"home/user/data", "/home/user/"},
		{directory.RootPath.String(), directory.NilParentPath.String()},
	} {
		t.Run(fmt.Sprintf("should return parent path for %s", tc.Path), func(t *testing.T) {
			// Given
			p := directory.NewPath(tc.Path)

			// When
			res := p.ParentPath()

			// Then
			assert.Equal(t, tc.Expected, res.String())
		})
	}
}

func TestPath_NewSubPath(t *testing.T) {
	for _, tc := range []struct {
		In         string
		NewSubPath string
		Expected   string
	}{
		{"/home/user/data/", "project", "/home/user/data/project/"},
		{"home/user/data", "project", "/home/user/data/project/"},
		{"", "", "/"},
		{directory.RootPath.String(), "user", "/user/"},
		{"/mydir/", "subdir", "/mydir/subdir/"},
	} {
		t.Run(fmt.Sprintf("should append %s to %s", tc.NewSubPath, tc.In), func(t *testing.T) {
			// Given
			p := directory.NewPath(tc.In)

			// When
			res := p.NewSubPath(tc.NewSubPath)

			// Then
			assert.Equal(t, tc.Expected, res.String())
		})
	}
}

func TestPath_Is(t *testing.T) {
	for _, tc := range []struct {
		Path     string
		Dir      *directory.Directory
		Expected bool
	}{
		{
			Path:     "/home/user/data/",
			Dir:      testutil.NewNotLoadedDirectory(t, "data", "/home/user/"),
			Expected: true,
		},
		{
			Path:     "/home/user/data/",
			Dir:      testutil.NewNotLoadedDirectory(t, "data2", "/home/user/"),
			Expected: false,
		},
	} {
		t.Run(fmt.Sprintf("should return %t for %s", tc.Expected, tc.Path), func(t *testing.T) {
			// Given
			p := directory.NewPath(tc.Path)

			// When
			res := p.Is(tc.Dir)

			// Then
			assert.Equal(t, tc.Expected, res)
		})
	}
}

func TestPath_Split(t *testing.T) {
	for _, tc := range []struct {
		Path  string
		Parts []string
	}{
		{
			Path:  "/home/user/data/",
			Parts: []string{"home", "user", "data"},
		},
		{
			Path:  "home/user/data",
			Parts: []string{"home", "user", "data"},
		},
		{
			Path:  "/",
			Parts: []string{""},
		},
		{
			Path:  "",
			Parts: []string{""},
		},
	} {
		t.Run(fmt.Sprintf("should return %v for %s", tc.Parts, tc.Path), func(t *testing.T) {
			// Given
			p := directory.NewPath(tc.Path)

			// When
			res := p.Split()

			// Then
			assert.Equal(t, tc.Parts, res)
		})
	}
}

func TestPath_RelativeTo(t *testing.T) {
	for _, tc := range []struct {
		Path, RelativeTo string
		Expected         string
	}{
		{
			Path:       "/home/user/data/",
			RelativeTo: "/home/",
			Expected:   "user/data/",
		},
		{
			Path:       "/home/",
			RelativeTo: "/home/",
			Expected:   "",
		},
		{
			Path:       "/",
			RelativeTo: "/",
			Expected:   "",
		},
		{
			Path:       "",
			RelativeTo: "",
			Expected:   "",
		},
	} {
		t.Run(fmt.Sprintf("should return %s for %s relative to %s", tc.Expected, tc.Path, tc.RelativeTo), func(t *testing.T) {
			// Given
			p := directory.NewPath(tc.Path)

			// When
			res, err := p.RelativeTo(directory.Path(tc.RelativeTo))

			// Then
			assert.NoError(t, err)
			assert.Equal(t, tc.Expected, res.String())
		})
	}

	t.Run("should return error when the base path is longer than the path", func(t *testing.T) {
		// Given
		p := directory.NewPath("/home/user/data/")

		// When
		_, err := p.RelativeTo("/home/user/data/projects/")

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, directory.ErrPathIncorrect)
		assert.ErrorContains(t, err, "base path is longer than the path")
	})

	t.Run("should return an error when the base path is not a parent of the path", func(t *testing.T) {
		// Given
		p := directory.NewPath("/home/user/data/")

		// When
		_, err := p.RelativeTo("/home/other/")

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, directory.ErrPathIncorrect)
		assert.ErrorContains(t, err, "base path is not a parent of the path")
	})
}
