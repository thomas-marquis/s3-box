package directory

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound  = errors.New("object not found in directory")
	ErrTechnical = errors.New("technical error occurred")
	ErrNotLoaded = errors.New("directory must be loaded first")
)

type Error struct {
	message string
	dir     *Directory
}

func NewError(dir *Directory, message string) error {
	return &Error{message: message, dir: dir}
}

func (e *Error) Error() string {
	return fmt.Sprintf("directory (%s) error: %s", e.dir.Path(), e.message)
}
