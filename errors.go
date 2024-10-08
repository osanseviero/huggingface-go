package huggingface

import (
	"fmt"
)

// APIError represents an error returned by the Hugging Face API
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Hugging Face API Error %d: %s", e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}