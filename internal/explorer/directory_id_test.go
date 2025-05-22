package explorer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/explorer"
)

func Test_ToName_ShouldReturnNameOfDirectory(t *testing.T) {
	testCases := []struct {
		id           explorer.S3DirectoryID
		expectedName string
	}{
		{explorer.RootDirID, ""},
		{explorer.S3DirectoryID("/"), ""},
		{explorer.S3DirectoryID("/path/"), "path"},
		{explorer.S3DirectoryID("/path/to/dir/"), "dir"},
		{explorer.S3DirectoryID("/path/to/dir/subdir/subsubdir/"), "subsubdir"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id.String(), func(t *testing.T) {
			// When
			name := testCase.id.ToName()

			// Then
			assert.Equal(t, testCase.expectedName, name)
		})
	}
}

func Test_InferParentID_ShouldReturnCorrectParentID(t *testing.T) {
	testCases := []struct {
		id               explorer.S3DirectoryID
		expectedParentID explorer.S3DirectoryID
	}{
		{explorer.RootDirID, explorer.NilParentID},
		{explorer.NilParentID, explorer.NilParentID},
		{explorer.S3DirectoryID("/path/"), explorer.RootDirID},
		{explorer.S3DirectoryID("/path/to/"), explorer.S3DirectoryID("/path/")},
		{explorer.S3DirectoryID("/path/to/dir/"), explorer.S3DirectoryID("/path/to/")},
		{explorer.S3DirectoryID("/path/to/dir/subdir/subsubdir/"), explorer.S3DirectoryID("/path/to/dir/subdir/")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.id.String(), func(t *testing.T) {
			// When
			name := testCase.id.InferParentID()

			// Then
			assert.Equal(t, testCase.expectedParentID, name)
		})
	}
}
