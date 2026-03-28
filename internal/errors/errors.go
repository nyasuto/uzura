// Package errors defines sentinel errors and structured error types for Uzura.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
var (
	// ErrRobotsDisallowed indicates the URL is blocked by robots.txt.
	ErrRobotsDisallowed = errors.New("blocked by robots.txt")

	// ErrTooManyRedirects indicates the redirect limit was exceeded.
	ErrTooManyRedirects = errors.New("too many redirects")

	// ErrInvalidSelector indicates a malformed CSS selector string.
	ErrInvalidSelector = errors.New("invalid CSS selector")
)

// FetchError represents an HTTP fetch failure with status and URL context.
type FetchError struct {
	StatusCode int
	URL        string
	Err        error
}

// Error implements the error interface.
func (e *FetchError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("fetch %s: HTTP %d", e.URL, e.StatusCode)
	}
	return fmt.Sprintf("fetch %s: %v", e.URL, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *FetchError) Unwrap() error {
	return e.Err
}
