package opencorpora

import (
	"fmt"
)

type OpenCorporaError struct {
	source  string
	wrapped error
}

func (e *OpenCorporaError) Unwrap() error {
	return e.wrapped
}

func (e *OpenCorporaError) Error() string {
	if e.wrapped != nil {
		return e.source + ": " + e.wrapped.Error()
	}

	return e.source + ": unspecified"
}

func NewErrorf(format string, args ...interface{}) *OpenCorporaError {
	source := fmt.Sprintf(format, args...)
	return &OpenCorporaError{source: source, wrapped: nil}
}

func NewOpenCorporaError(source string, wrap error) *OpenCorporaError {
	return &OpenCorporaError{source: source, wrapped: wrap}
}

func WrapOpenCorporaErrorf(wrap error, format string, args ...interface{}) *OpenCorporaError {
	source := fmt.Sprintf(format, args...)
	return &OpenCorporaError{source: source, wrapped: wrap}
}

func WrapOpenCorporaError(wrap error, source string) *OpenCorporaError {
	return &OpenCorporaError{source: source, wrapped: wrap}
}
