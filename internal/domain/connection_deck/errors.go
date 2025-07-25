package connection_deck

import "errors"

var (
	ErrNotFound  = errors.New("connection not found")
	ErrTechnical = errors.New("technical error occurred while processing the connection")
)
