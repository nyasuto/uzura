package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		wrapped  error
	}{
		{
			name:     "ErrRobotsDisallowed",
			sentinel: ErrRobotsDisallowed,
			wrapped:  fmt.Errorf("%w: http://example.com/secret", ErrRobotsDisallowed),
		},
		{
			name:     "ErrTooManyRedirects",
			sentinel: ErrTooManyRedirects,
			wrapped:  fmt.Errorf("%w: stopped after 10", ErrTooManyRedirects),
		},
		{
			name:     "ErrInvalidSelector",
			sentinel: ErrInvalidSelector,
			wrapped:  fmt.Errorf("%w: parse error", ErrInvalidSelector),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.wrapped, tt.sentinel)
			}
		})
	}
}

func TestFetchError(t *testing.T) {
	t.Run("with status code", func(t *testing.T) {
		err := &FetchError{
			StatusCode: 404,
			URL:        "http://example.com/missing",
		}
		want := "fetch http://example.com/missing: HTTP 404"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("with underlying error", func(t *testing.T) {
		inner := fmt.Errorf("connection refused")
		err := &FetchError{
			URL: "http://example.com",
			Err: inner,
		}
		want := "fetch http://example.com: connection refused"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		inner := ErrTooManyRedirects
		err := &FetchError{
			URL: "http://example.com",
			Err: inner,
		}
		if !errors.Is(err, ErrTooManyRedirects) {
			t.Error("errors.Is(FetchError, ErrTooManyRedirects) = false, want true")
		}
	})

	t.Run("errors.As", func(t *testing.T) {
		err := fmt.Errorf("loading: %w", &FetchError{
			StatusCode: 500,
			URL:        "http://example.com",
		})
		var fe *FetchError
		if !errors.As(err, &fe) {
			t.Fatal("errors.As failed")
		}
		if fe.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", fe.StatusCode)
		}
		if fe.URL != "http://example.com" {
			t.Errorf("URL = %q, want %q", fe.URL, "http://example.com")
		}
	})
}
