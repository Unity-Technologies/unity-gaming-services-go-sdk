package localproxy

import "fmt"

// UnexpectedResponseError represents an unexpected response from the local proxy.
type UnexpectedResponseError struct {
	RequestID    string
	StatusCode   int
	ResponseBody string
}

// NewUnexpectedResponseWithBody creates a new UnexpectedResponseError from a response body.
func NewUnexpectedResponseWithBody(requestID string, statusCode int, responseBody []byte) *UnexpectedResponseError {
	return &UnexpectedResponseError{
		RequestID:    requestID,
		StatusCode:   statusCode,
		ResponseBody: string(responseBody),
	}
}

// NewUnexpectedResponseWithError creates a new UnexpectedResponseError from an error.
func NewUnexpectedResponseWithError(requestID string, statusCode int, err error) *UnexpectedResponseError {
	return &UnexpectedResponseError{
		RequestID:    requestID,
		StatusCode:   statusCode,
		ResponseBody: err.Error(),
	}
}

// Error returns the string representation of the error.
func (e *UnexpectedResponseError) Error() string {
	return fmt.Sprintf("unexpected response from local proxy, request ID: %s, status: %d, error: %s", e.RequestID, e.StatusCode, e.ResponseBody)
}
