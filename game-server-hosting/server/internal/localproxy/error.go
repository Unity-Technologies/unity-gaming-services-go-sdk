package localproxy

import "github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"

// NewUnexpectedResponseWithBody creates a new UnexpectedResponseError from a response body.
func NewUnexpectedResponseWithBody(requestID string, statusCode int, responseBody []byte) *model.UnexpectedResponseError {
	return &model.UnexpectedResponseError{
		RequestID:    requestID,
		StatusCode:   statusCode,
		ResponseBody: string(responseBody),
	}
}

// NewUnexpectedResponseWithError creates a new UnexpectedResponseError from an error.
func NewUnexpectedResponseWithError(requestID string, statusCode int, err error) *model.UnexpectedResponseError {
	return &model.UnexpectedResponseError{
		RequestID:    requestID,
		StatusCode:   statusCode,
		ResponseBody: err.Error(),
	}
}
