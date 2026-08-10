package directory

import (
	"errors"
	"io"
)

var (
	ErrInvalidSeek = errors.New("invalid seek")
)

type Canceler interface {
	// Cancel cancels all in-progress operations.
	// This method can leave the underlying object in an inconsistent state.
	Cancel()
}

type FileContent interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	Canceler
}

type InMemoryContent struct {
	Data []byte
	Pos  int64
}

func (f *InMemoryContent) Read(p []byte) (int, error) {
	if f.Pos >= int64(len(f.Data)) {
		return 0, io.EOF
	}
	n := copy(p, f.Data[f.Pos:])
	f.Pos += int64(n)
	return n, nil
}

func (f *InMemoryContent) Write(p []byte) (int, error) {
	f.Data = append(f.Data, p...)
	f.Pos += int64(len(p))
	return len(p), nil
}

func (f *InMemoryContent) Close() error {
	return nil
}

func (f *InMemoryContent) Seek(offset int64, whence int) (int64, error) {
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

func (f *InMemoryContent) Cancel() {}

//
//type InMemContentActionable struct {
//	*InMemoryContent
//	actRead chan struct{}
//}
//
//func NewInMemContentActionable() *InMemContentActionable {
//	c := &InMemContentActionable{}
//	return c
//}
//
//func (f *InMemContentActionable) Read(p []byte) (int, error) {
//	return f.InMemoryContent.Read(p)
//}
//
//func (f *InMemContentActionable) Write(p []byte) (int, error) {
//	return f.InMemoryContent.Write(p)
//}
//
//func (f *InMemContentActionable) Close() error {
//	return f.InMemoryContent.Close()
//}
//
//func (f *InMemContentActionable) Seek(offset int64, whence int) (int64, error) {
//	return f.InMemoryContent.Seek(offset, whence)
//}
//
//func (f *InMemContentActionable) Cancel() {
//	f.InMemoryContent.Cancel()
//}
