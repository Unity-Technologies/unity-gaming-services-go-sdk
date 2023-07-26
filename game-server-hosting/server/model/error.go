package model

import "fmt"

// UnexpectedResponseError represents an unexpected response from the local proxy.
type UnexpectedResponseError struct {
	RequestID    string
	StatusCode   int
	ResponseBody string
}

// Error returns the string representation of the error.
func (e *UnexpectedResponseError) Error() string {
	return fmt.Sprintf("unexpected response from local proxy, request ID: %s, status: %d, error: %s", e.RequestID, e.StatusCode, e.ResponseBody)
}
