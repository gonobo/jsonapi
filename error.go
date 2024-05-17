package jsonapi

import (
	"errors"
	"fmt"
)

var (
	ErrJSONAPI = errors.New("jsonapi error")
)

func jsonapiError(format string, v ...any) error {
	msg := fmt.Sprintf(format, v...)
	return fmt.Errorf("%w: %s", ErrJSONAPI, msg)
}

// NewError creates a new ErrorNode with the given status and title.
func NewError(cause error, title string) Error {
	return Error{
		Title:  title,
		Detail: cause.Error(),
	}
}
