package directory

type Option func(*Directory)

func WithFiles(files []*File) Option {
	return func(d *Directory) {
		d.files = files
	}
}

func WithSubDirectories(subDirs []Path) Option {
	return func(d *Directory) {
		d.subDirectories = subDirs
	}
}
