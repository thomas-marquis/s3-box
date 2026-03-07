package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func TestUpdateObjectKey(t *testing.T) {
	tcs := []struct {
		dir                             *directory.Directory
		oldKey, newDirName, expectedKey string
	}{
		{
			dir:         testutil.NewDirectory(t, "mydir", directory.RootPath),
			oldKey:      "mydir/file.txt",
			newDirName:  "newdir",
			expectedKey: "newdir/file.txt",
		},
		{
			dir:         testutil.NewDirectory(t, "mydir", directory.RootPath),
			oldKey:      "mydir/subdir/nested/file.txt",
			newDirName:  "newdir",
			expectedKey: "newdir/subdir/nested/file.txt",
		},
		{
			dir:         testutil.NewDirectory(t, "mydir", directory.NewPath("base")),
			oldKey:      "base/mydir/file.txt",
			newDirName:  "newdir",
			expectedKey: "base/newdir/file.txt",
		},
		{
			dir:         testutil.NewDirectory(t, "mydir", directory.NewPath("base")),
			oldKey:      "base/mydir/mydir/mydir/file.txt",
			newDirName:  "newdir",
			expectedKey: "base/newdir/mydir/mydir/file.txt",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.expectedKey, func(t *testing.T) {
			assert.Equal(t, tc.expectedKey, updateObjectKey(tc.dir, tc.oldKey, tc.newDirName))
		})
	}
}
