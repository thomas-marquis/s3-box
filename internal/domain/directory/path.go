package directory

import (
	"errors"
	"strings"
)

var ErrPathIncorrect = errors.New("path is incorrect")

type Path string

// NewPath create a new absolute path from a string.
func NewPath(path string) Path {
	if path == "" {
		return NilParentPath
	}
	if path == "/" {
		return RootPath
	}
	pathContent := path
	if !strings.HasSuffix(path, "/") {
		pathContent += "/"
	}
	if !strings.HasPrefix(pathContent, "/") {
		pathContent = "/" + pathContent
	}
	return Path(pathContent)
}

func (p Path) String() string {
	return string(p)
}

func (p Path) DirectoryName() string {
	if p == RootPath {
		return RootDirName
	}
	pathStriped := strings.TrimSuffix(p.String(), "/")
	pathSplit := strings.Split(pathStriped, "/")
	dirName := pathSplit[len(pathSplit)-1]
	return dirName
}

func (p Path) ParentPath() Path {
	if p == RootPath || p == NilParentPath {
		return NilParentPath
	}
	pathStriped := strings.TrimSuffix(p.String(), "/")
	pathSplit := strings.Split(pathStriped, "/")
	if len(pathSplit) <= 1 {
		return RootPath
	}
	pathSplit = pathSplit[:len(pathSplit)-1]
	parentPath := strings.Join(pathSplit, "/")
	if parentPath == "" {
		return RootPath
	}
	return Path(parentPath + "/")
}

func (p Path) NewSubPath(name string) Path {
	if p == NilParentPath && name == "" {
		return RootPath
	}
	return NewPath(p.String() + name)
}

func (p Path) Is(dir *Directory) bool {
	if dir == nil {
		return false
	}
	return dir.Path() == p
}

func (p Path) Split() []string {
	return strings.Split(
		strings.Trim(p.String(), "/"),
		"/")
}

// RelativeTo returns the path relative to the base path.
// The base path must be a parent of the path.
func (p Path) RelativeTo(basePath Path) (Path, error) {
	if basePath == RootPath || basePath == NilParentPath {
		return "", nil
	}

	pSPlited := p.Split()
	basePathSPlited := basePath.Split()
	l := len(basePathSPlited)
	if l > len(pSPlited) {
		return "", errors.Join(ErrPathIncorrect, errors.New("base path is longer than the path"))
	}
	for i, pS := range pSPlited {
		if i >= l {
			break
		}
		if pS != basePathSPlited[i] {
			return "", errors.Join(ErrPathIncorrect, errors.New("base path is not a parent of the path"))
		}
	}

	res := pSPlited[l:]
	if len(res) == 0 {
		return "", nil
	}

	return Path(
		strings.TrimPrefix(
			strings.Join(res, "/"),
			"/") + "/"), nil
}
