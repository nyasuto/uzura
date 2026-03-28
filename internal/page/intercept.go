package page

import (
	"context"
	"net/http"
)

// InterceptAction is the decision made by an interceptor on a paused request.
type InterceptAction int

const (
	// InterceptContinue allows the request to proceed normally.
	InterceptContinue InterceptAction = iota
	// InterceptFail aborts the request with the given error reason.
	InterceptFail
)

// InterceptedRequest holds information about a request that has been paused.
type InterceptedRequest struct {
	RequestID string
	URL       string
	Method    string
	Headers   http.Header
}

// InterceptResult is the decision returned by a RequestInterceptor.
type InterceptResult struct {
	Action      InterceptAction
	ErrorReason string // CDP Network.ErrorReason (e.g. "Failed", "Aborted", "BlockedByClient")
}

// RequestInterceptor is called before each HTTP request.
// The interceptor may block until a decision is made (e.g. waiting for a CDP
// client to call Fetch.continueRequest or Fetch.failRequest).
// Returning nil is equivalent to InterceptContinue.
type RequestInterceptor func(ctx context.Context, req InterceptedRequest) (*InterceptResult, error)
