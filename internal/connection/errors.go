package connection

import "errors"

var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrConnectionFailed   = errors.New("s3 server connection error")
)
