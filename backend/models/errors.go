package models

import "fmt"

type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewNotFoundError(resource string) *APIError {
	return &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("%s not found", resource),
	}
}

func NewBadRequestError(message string) *APIError {
	return &APIError{
		StatusCode: 400,
		Message:    message,
	}
}

func NewConflictError(message string) *APIError {
	return &APIError{
		StatusCode: 409,
		Message:    message,
	}
}
