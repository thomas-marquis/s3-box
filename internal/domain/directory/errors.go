package directory

import "errors"

var (
	ErrNotFound  = errors.New("objecto not found in directory")
	ErrTechnical = errors.New("technical error occurred")
)
