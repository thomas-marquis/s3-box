package explorer

import "errors"

var (
	ErrObjectNotFound  = errors.New("object not found")
	ErrConnectionNoSet = errors.New("connection not set")
)
