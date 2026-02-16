package adminapi

import (
	"errors"
	"fmt"
)

var (
	// ErrNoResults is returned by One() when the query matches zero objects.
	ErrNoResults = errors.New("no server objects found")

	// ErrMultipleResults is returned by One() when the query matches more than one object.
	ErrMultipleResults = errors.New("expected exactly one server object, got multiple")

	// ErrUnknownAttribute is returned by Set() when the attribute does not exist on the object.
	ErrUnknownAttribute = errors.New("unknown attribute")
)

// APIError represents an HTTP error response from the Serveradmin API.
// Use errors.As() to inspect status codes and messages from API failures.
type APIError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP error %d %s: %s", e.StatusCode, e.Status, e.Message)
	}
	return fmt.Sprintf("HTTP error %d %s", e.StatusCode, e.Status)
}
