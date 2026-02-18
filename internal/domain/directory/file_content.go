package directory

import (
	"errors"
	"io"
	"os"
)

var (
	ErrContentReading = errors.New("error reading file content")
	ErrContentWriting = errors.New("error writing file content")
)

// TODO: remove this struct
type Content struct {
	file                 *File
	fileObj              *os.File
	filePtah             string
	hasBeenAlreadyOpened bool
	openMode             fileOpenMode
}

type ContentOption func(*Content)

type fileOpenMode int

const (
	openModeRead fileOpenMode = iota
	openModeWrite
)

func FromLocalFile(path string) ContentOption {
	return func(c *Content) {
		c.filePtah = path
	}
}

func ContentFromFile(obj *os.File) ContentOption {
	return func(c *Content) {
		c.fileObj = obj
	}
}

func WithOpenModeRead() ContentOption {
	return func(c *Content) {
		c.openMode = openModeRead
	}
}

func WithOpenModeWrite() ContentOption {
	return func(c *Content) {
		c.openMode = openModeWrite
	}
}

func NewFileContent(file *File, opts ...ContentOption) *Content {
	c := &Content{file: file, openMode: openModeRead}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(c)
	}
	return c
}

func (c *Content) Open() (*os.File, error) {
	if c.filePtah != "" {
		if c.hasBeenAlreadyOpened {
			return nil, errors.Join(ErrContentReading, errors.New("file content has already been opened"))
		}
		_, err := os.Stat(c.filePtah)
		var f *os.File
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Join(ErrContentReading, err)
		}

		switch c.openMode {
		case openModeWrite:
			f, err = os.OpenFile(c.filePtah, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return nil, errors.Join(ErrContentWriting, err)
			}
		default:
			if err != nil && os.IsNotExist(err) {
				return nil, errors.Join(ErrContentReading, err)
			}
			f, err = os.Open(c.filePtah)
			if err != nil {
				return nil, errors.Join(ErrContentReading, err)
			}
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return nil, errors.Join(ErrContentReading, err)
			}
		}
		c.fileObj = f
		c.hasBeenAlreadyOpened = true
		return f, nil
	}
	if c.fileObj == nil {
		return nil, errors.Join(ErrContentReading, errors.New("file content is nil"))
	}
	return c.fileObj, nil
}

func (c *Content) File() *File {
	return c.file
}

// TODO: rename to Content
type FileObject interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
}

var ErrInvalidSeek = errors.New("invalid seek")

type InMemoryFileObject struct {
	Data []byte
	Pos  int64
}

func (f *InMemoryFileObject) Read(p []byte) (int, error) {
	if f.Pos >= int64(len(f.Data)) {
		return 0, io.EOF
	}
	n := copy(p, f.Data[f.Pos:])
	f.Pos += int64(n)
	return n, nil
}

func (f *InMemoryFileObject) Write(p []byte) (int, error) {
	f.Data = append(f.Data, p...)
	f.Pos += int64(len(p))
	return len(p), nil
}

func (f *InMemoryFileObject) Close() error {
	return nil
}

func (f *InMemoryFileObject) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = f.Pos + offset
	case io.SeekEnd:
		newPos = int64(len(f.Data)) + offset
	default:
		return 0, ErrInvalidSeek
	}
	if newPos < 0 {
		return 0, ErrInvalidSeek
	}
	f.Pos = newPos
	return f.Pos, nil
}
