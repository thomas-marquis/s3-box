package directory

import "time"

type FileOption func(*File)

func WithFileSize(sizeBytes int64) FileOption {
	return func(f *File) {
		f.sizeBytes = sizeBytes
	}
}

func WithFileLastModified(lastModified time.Time) FileOption {
	return func(f *File) {
		f.lastModified = lastModified
	}
}
