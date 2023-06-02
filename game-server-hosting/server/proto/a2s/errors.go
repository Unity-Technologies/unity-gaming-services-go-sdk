package a2s

import (
	"errors"
	"fmt"
)

type (
	// UnsupportedQueryError is an error which represents an invalid SQP query header.
	UnsupportedQueryError struct {
		header []byte
	}
)

// errNotAnInfoRequest defines an error in which the input is not an A2S_INFO request.
var errNotAnInfoRequest = errors.New("not an info request")

// NewUnsupportedQueryError returns a new instance of UnsupportedQueryError.
func NewUnsupportedQueryError(header []byte) error {
	return &UnsupportedQueryError{
		header: header,
	}
}

// Error returns the error string.
func (e *UnsupportedQueryError) Error() string {
	return fmt.Sprintf("unsupported query: %x", e.header)
}
