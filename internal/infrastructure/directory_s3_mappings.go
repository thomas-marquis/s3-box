package infrastructure

import (
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"strings"
)

func mapDirToObjectKey(dir *directory.Directory) string {
	if dir.Path().String() == "" || dir.IsRoot() {
		return ""
	}
	return strings.TrimPrefix(dir.Path().String(), "/")
}

func mapPathToSearchKey(path directory.Path) string {
	if path.String() == "" || path == directory.RootPath {
		return ""
	}
	key := strings.TrimPrefix(path.String(), "/")
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	return key
}

func mapFileToKey(file *directory.File) string {
	return strings.TrimPrefix(file.FullPath(), "/")
}

func mapKeyToObjectName(key string) string {
	if key == "" || key == "/" {
		return ""
	}
	dirPathStriped := strings.TrimSuffix(key, "/")
	dirPathSplit := strings.Split(dirPathStriped, "/")
	return dirPathSplit[len(dirPathSplit)-1]
}
