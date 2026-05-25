package kapso

import "fmt"

type APIError struct {
	StatusCode int
	Message    string
	RetryAfter int
}

type RateLimitError struct {
	APIError
}

type NotFoundError struct {
	APIError
}

type ValidationError struct {
	APIError
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited (%d): %s", e.StatusCode, e.Message)
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found (%d): %s", e.StatusCode, e.Message)
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error (%d): %s", e.StatusCode, e.Message)
}
