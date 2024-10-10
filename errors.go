package huggingface

import (
	"fmt"
	"io"
	"net/http"
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

// createApiError creates a descriptive error message from the API response.
func CreateApiError(resp *http.Response) error {
	responseBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API request failed: status code %d, body: %s", resp.StatusCode, string(responseBody))
}