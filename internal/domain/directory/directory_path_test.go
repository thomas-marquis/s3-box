package directory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func Test_DirectoryName_ShouldReturnNameOfDirectory(t *testing.T) {
	testCases := []struct {
		path         directory.Path
		expectedName string
	}{
		{directory.RootPath, ""},
		{directory.NewPath("/"), ""},
		{directory.NewPath("/path/"), "path"},
		{directory.NewPath("/path"), "path"},
		{directory.NewPath("/path/to/dir/"), "dir"},
		{directory.NewPath("/path/to/dir/subdir/subsubdir/"), "subsubdir"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.path.String(), func(t *testing.T) {
			// When
			name := testCase.path.DirectoryName()

			// Then
			assert.Equal(t, testCase.expectedName, name)
		})
	}
}

func Test_ParentPath_ShouldReturnCorrectParentPath(t *testing.T) {
	testCases := []struct {
		path             directory.Path
		expectedParentID directory.Path
	}{
		{directory.RootPath, directory.NilParentPath},
		{directory.NilParentPath, directory.NilParentPath},
		{directory.NewPath("/path/"), directory.RootPath},
		{directory.NewPath("/path"), directory.RootPath},
		{directory.NewPath("/path/to/"), directory.NewPath("/path/")},
		{directory.NewPath("/path/to/dir/"), directory.NewPath("/path/to/")},
		{directory.NewPath("/path/to/dir/subdir/subsubdir/"), directory.NewPath("/path/to/dir/subdir/")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.path.String(), func(t *testing.T) {
			// When
			name := testCase.path.ParentPath()

			// Then
			assert.Equal(t, testCase.expectedParentID, name)
		})
	}
}
