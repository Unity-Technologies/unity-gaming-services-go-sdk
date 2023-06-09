package sqp

import (
	"errors"
	"fmt"
)

type (
	// UnsupportedSQPVersionError is an error which represents an invalid SQP version provided to the reader.
	UnsupportedSQPVersionError struct {
		version uint16
	}
)

var (
	errInvalidPacketLength = errors.New("invalid packet length")
	errUnsupportedQuery    = errors.New("unsupported query")
)

// NewUnsupportedSQPVersionError returns a new instance of UnsupportedSQPVersionError.
func NewUnsupportedSQPVersionError(version uint16) error {
	return &UnsupportedSQPVersionError{
		version: version,
	}
}

// Error returns the error string.
func (e *UnsupportedSQPVersionError) Error() string {
	return fmt.Sprintf("unsupported sqp version: %d", e.version)
}
