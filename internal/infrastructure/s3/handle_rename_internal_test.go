package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetObjectDstKey(t *testing.T) {
	tcs := []struct {
		srcDirKey, dstDirKey, oldKey, expected string
	}{
		{
			srcDirKey: "mydir/",
			dstDirKey: "newdir/",
			oldKey:    "mydir/file.txt",
			expected:  "newdir/file.txt",
		},
		{
			srcDirKey: "mydir/",
			dstDirKey: "newdir/",
			oldKey:    "mydir/subdir/nested/file.txt",
			expected:  "newdir/subdir/nested/file.txt",
		},
		{
			srcDirKey: "base/mydir/",
			dstDirKey: "base/newdir/",
			oldKey:    "base/mydir/file.txt",
			expected:  "base/newdir/file.txt",
		},
		{
			srcDirKey: "base/mydir/",
			dstDirKey: "base/newdir/",
			oldKey:    "base/mydir/mydir/mydir/file.txt",
			expected:  "base/newdir/mydir/mydir/file.txt",
		},
		{
			srcDirKey: "base/mydir/",
			dstDirKey: "base/newdir/",
			oldKey:    "base/mydir/base/mydir/file.txt",
			expected:  "base/newdir/base/mydir/file.txt",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, getObjectDstKey(tc.srcDirKey, tc.dstDirKey, tc.oldKey))
		})
	}
}

func TestGetDstKey(t *testing.T) {
	tcs := []struct {
		srcDrKey, newName, expected string
	}{
		{
			srcDrKey: "mydir/",
			newName:  "newdir",
			expected: "newdir/",
		},
		{
			srcDrKey: "base/mydir/",
			newName:  "newdir",
			expected: "base/newdir/",
		},
		{
			srcDrKey: "mydir/base/mydir/",
			newName:  "newdir",
			expected: "mydir/base/newdir/",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, getDstDirKey(tc.srcDrKey, tc.newName))
		})
	}
}
