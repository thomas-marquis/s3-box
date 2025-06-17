package directory

type DirectoryOption func(*Directory)

func WithFiles(files []*File) DirectoryOption {
	return func(d *Directory) {
		d.files = files
	}
}

func WithSubDirectories(subDirs []Path) DirectoryOption {
	return func(d *Directory) {
		d.subDirectories = subDirs
	}
}
