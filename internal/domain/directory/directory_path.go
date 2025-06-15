package directory

import "strings"

type Path string

func NewPath(path string) Path {
	if path == "" {
		return NilParentPath
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return Path(path)
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
	return NewPath(p.String() + name)
}
