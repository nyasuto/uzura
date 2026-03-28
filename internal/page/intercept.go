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
	// InterceptFulfill provides a mock response without making the actual request.
	InterceptFulfill
)

// InterceptStage indicates at which point the request was paused.
type InterceptStage int

const (
	// StageRequest means the request was paused before being sent.
	StageRequest InterceptStage = iota
	// StageResponse means the request was paused after the response arrived.
	StageResponse
)

// InterceptedRequest holds information about a request that has been paused.
type InterceptedRequest struct {
	RequestID string
	URL       string
	Method    string
	Headers   http.Header
	Stage     InterceptStage

	// Response fields (populated when Stage == StageResponse).
	StatusCode      int
	ResponseHeaders http.Header
	Body            []byte
}

// InterceptResult is the decision returned by a RequestInterceptor.
type InterceptResult struct {
	Action      InterceptAction
	ErrorReason string // CDP Network.ErrorReason (e.g. "Failed", "Aborted", "BlockedByClient")

	// Request overrides (used with InterceptContinue at request stage).
	URL     string      // Override URL (empty = no change).
	Headers http.Header // Override headers (nil = no change).

	// Fulfill fields (used with InterceptFulfill).
	ResponseCode    int
	ResponseHeaders map[string]string
	ResponseBody    []byte

	// Response overrides (used with InterceptContinue at response stage).
	RespStatusCode int
	RespHeaders    map[string]string
}

// RequestInterceptor is called before each HTTP request and optionally after
// the response arrives. The interceptor may block until a decision is made.
// Returning nil is equivalent to InterceptContinue.
type RequestInterceptor func(ctx context.Context, req InterceptedRequest) (*InterceptResult, error)
