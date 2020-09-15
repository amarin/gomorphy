package grammemes

import (
	"errors"
	"fmt"
)

var (
	ErrDecode = errors.New("unmarshal")
)

// Error type implements error interface and used in package routines.
type Error struct {
	formattedString string
	wrapped         error
}

// Error returns formatted error string. Implements error interface.
func (e Error) Error() string {
	if e.wrapped != nil {
		return e.formattedString + ": " + e.wrapped.Error()
	}

	return e.formattedString
}

// GoString returns formatted error string prefixes with package name and type. Implements GoStringer.
func (e Error) GoString() string {
	return "grammemes.Error:" + e.Error()
}

// NewErrorf is a default error constructor.
// Takes format and arguments to make error description using fmt.Sprintf() formatting rules.
func NewErrorf(format string, args ...interface{}) error {
	return &Error{formattedString: fmt.Sprintf(format, args...)}
}

// WrapErrorf wraps underlying error into new text error
// Takes format and arguments to make error description using fmt.Sprintf() formatting rules.
func WrapErrorf(wrap error, format string, args ...interface{}) *Error {
	return &Error{formattedString: fmt.Sprintf(format, args...), wrapped: wrap}
}
