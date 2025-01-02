package viewerror

import "errors"

var (
	ErrNoConnectionSelected = errors.New("no connection selected")
	ErrConnectionFailed     = errors.New("connection to s3 server failed")
)
