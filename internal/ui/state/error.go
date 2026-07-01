package state

import (
	"errors"
)

var (
	ErrState = errors.New("state error")
)

type Error struct {
	message string
	wrapped []error
}

func NewError(message string, wrapped ...error) Error {
	return Error{message: message, wrapped: wrapped}
}

func (e Error) Error() string {
	return e.message
}

func (e Error) Unwrap() []error {
	errs := []error{
		ErrState,
		e,
	}
	return append(errs, e.wrapped...)
}
