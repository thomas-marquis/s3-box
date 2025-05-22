package explorer

import (
	"strings"
)

type S3DirectoryID string

func (id S3DirectoryID) String() string {
	return string(id)
}

func (id S3DirectoryID) ToName() string {
	if id == RootDirID {
		return RootDirName
	}
	dirPathStriped := strings.TrimSuffix(id.String(), "/")
	dirPathSplit := strings.Split(dirPathStriped, "/")
	dirName := dirPathSplit[len(dirPathSplit)-1]
	return dirName
}

func (id S3DirectoryID) InferParentID() S3DirectoryID {
	if id == RootDirID || id == NilParentID {
		return NilParentID
	}
	dirPathSplit := strings.Split(id.String(), "/")
	if len(dirPathSplit) <= 3 {
		return RootDirID
	}
	dirPathSplit = dirPathSplit[:len(dirPathSplit)-1]
	dirPathSplit[len(dirPathSplit)-1] = ""
	parentID := strings.Join(dirPathSplit, "/")

	return S3DirectoryID(parentID)
}
