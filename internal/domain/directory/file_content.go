package directory

import (
	"errors"
	"os"
)

var (
	ErrContentReading = errors.New("error reading file content")
	ErrContentWriting = errors.New("error writing file content")
)

type Content struct {
	file                 *File
	fileObj              *os.File
	filePtah             string
	hasBeenAlreadyOpened bool
}

type ContentOption func(*Content)

func FromLocalFile(path string) ContentOption {
	return func(c *Content) {
		c.filePtah = path
	}
}

func FromFileObj(obj *os.File) ContentOption {
	return func(c *Content) {
		c.fileObj = obj
	}
}

func NewFileContent(file *File, opt ContentOption) *Content {
	c := &Content{file: file}
	if opt != nil {
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
		if os.IsExist(err) {
			f, err = os.Open(c.filePtah)
			if err != nil {
				return nil, errors.Join(ErrContentReading, err)
			}
		} else {
			f, err = os.Create(c.filePtah)
			if err != nil {
				return nil, errors.Join(ErrContentWriting, err)
			}
		}
		c.fileObj = f
		c.hasBeenAlreadyOpened = true
		return f, nil
	}
	return c.fileObj, nil
}

func (c *Content) File() *File {
	return c.file
}
