// Package clickup — shared domain models and error types.
package clickup

import "fmt"

// APIError represents a non-2xx response from the ClickUp API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("clickup api error %d: %s", e.StatusCode, e.Body)
}

// User represents a ClickUp user (abbreviated).
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Team represents a ClickUp workspace (called "team" in the v2 API).
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
