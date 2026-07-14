package js

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// HTTPRequest is a network request issued by JS (fetch / XMLHttpRequest).
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// HTTPResponse is the result of an HTTPRequest. Body is fully buffered,
// decompressed and decoded by the injected client.
type HTTPResponse struct {
	Status     int
	StatusText string
	Headers    http.Header
	Body       []byte
	FinalURL   string
}

// HTTPClient performs network requests on behalf of JS bindings.
// uzura applies NO same-origin policy / CORS checks: it is an agent
// browser and does not carry a human user's credentials across sites.
type HTTPClient func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error)

// ErrNoHTTPClient is returned when JS attempts network access before a
// client has been injected (e.g. VM used standalone in tests).
var ErrNoHTTPClient = errors.New("js: network access not available (no HTTPClient injected)")

// SetHTTPClient injects the network client used by fetch and XMLHttpRequest.
func (vm *VM) SetHTTPClient(c HTTPClient) { vm.client = c }

func (vm *VM) httpClient() HTTPClient {
	if vm.client != nil {
		return vm.client
	}
	return func(context.Context, HTTPRequest) (*HTTPResponse, error) {
		return nil, ErrNoHTTPClient
	}
}

// SetBaseURL sets the document URL used to resolve relative request URLs.
func (vm *VM) SetBaseURL(raw string) {
	if u, err := url.Parse(raw); err == nil {
		vm.baseURL = u
	}
}

func (vm *VM) resolveURL(ref string) string {
	if vm.baseURL == nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return vm.baseURL.ResolveReference(r).String()
}
