package page

import (
	"context"
	"net/http"
)

// RequestHandler is called for each request before it is sent.
// The handler must call exactly one of req.Continue(), req.Abort(), or req.Fulfill().
type RequestHandler func(req *Request)

// ResponseHandler is called for each response before it is processed.
// The handler must call exactly one of resp.Continue() or resp.Fulfill().
type ResponseHandler func(resp *Response)

// Request represents an intercepted HTTP request (Go API).
// Exactly one of Continue, Abort, or Fulfill must be called.
type Request struct {
	URL       string
	Method    string
	Headers   http.Header
	RequestID string

	decided  bool
	decision InterceptResult
}

// Continue lets the request proceed, optionally with overrides.
func (r *Request) Continue(opts ...ContinueOption) {
	r.decided = true
	r.decision = InterceptResult{Action: InterceptContinue}
	for _, o := range opts {
		o.applyContinue(&r.decision)
	}
}

// Abort blocks the request with the given reason.
func (r *Request) Abort(reason string) {
	r.decided = true
	if reason == "" {
		reason = "BlockedByClient"
	}
	r.decision = InterceptResult{Action: InterceptFail, ErrorReason: reason}
}

// Fulfill provides a mock response without making the actual HTTP request.
func (r *Request) Fulfill(opts FulfillOption) {
	r.decided = true
	code := opts.Status
	if code == 0 {
		code = 200
	}
	r.decision = InterceptResult{
		Action:          InterceptFulfill,
		ResponseCode:    code,
		ResponseHeaders: opts.Headers,
		ResponseBody:    opts.Body,
	}
}

// Response represents an intercepted HTTP response (Go API).
type Response struct {
	URL        string
	Method     string
	StatusCode int
	Headers    http.Header
	Body       []byte
	RequestID  string

	decided  bool
	decision InterceptResult
}

// Continue lets the response proceed, optionally with header overrides.
func (r *Response) Continue(opts ...ResponseContinueOption) {
	r.decided = true
	r.decision = InterceptResult{Action: InterceptContinue}
	for _, o := range opts {
		o.applyResponseContinue(&r.decision)
	}
}

// Fulfill replaces the response body entirely.
func (r *Response) Fulfill(opts FulfillOption) {
	r.decided = true
	code := opts.Status
	if code == 0 {
		code = 200
	}
	r.decision = InterceptResult{
		Action:          InterceptFulfill,
		ResponseCode:    code,
		ResponseHeaders: opts.Headers,
		ResponseBody:    opts.Body,
	}
}

// ContinueOption modifies the request before it proceeds.
type ContinueOption interface {
	applyContinue(r *InterceptResult)
}

// WithURL overrides the request URL.
func WithURL(url string) ContinueOption { return urlOption(url) }

type urlOption string

func (u urlOption) applyContinue(r *InterceptResult) { r.URL = string(u) }

// WithHeaders overrides the request headers.
func WithHeaders(h http.Header) ContinueOption { return headerOption{h} }

type headerOption struct{ h http.Header }

func (o headerOption) applyContinue(r *InterceptResult) { r.Headers = o.h }

// ResponseContinueOption modifies the response before it is processed.
type ResponseContinueOption interface {
	applyResponseContinue(r *InterceptResult)
}

// WithResponseHeaders overrides response headers.
func WithResponseHeaders(h map[string]string) ResponseContinueOption {
	return respHeaderOption(h)
}

type respHeaderOption map[string]string

func (o respHeaderOption) applyResponseContinue(r *InterceptResult) {
	r.RespHeaders = map[string]string(o)
}

// FulfillOption configures a fulfilled response.
type FulfillOption struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

// OnRequest registers a Go callback for request interception.
// This sets up a RequestInterceptor that delegates to the handler at the
// request stage. It shares the same interception pipeline as the CDP Fetch domain.
func (p *Page) OnRequest(handler RequestHandler) {
	prev := p.requestInterceptor
	p.requestInterceptor = func(ctx context.Context, ireq InterceptedRequest) (*InterceptResult, error) {
		if ireq.Stage == StageRequest {
			req := &Request{
				URL:       ireq.URL,
				Method:    ireq.Method,
				Headers:   ireq.Headers,
				RequestID: ireq.RequestID,
			}
			handler(req)
			if req.decided {
				return &req.decision, nil
			}
		}
		// Fall through to existing interceptor (e.g. CDP FetchDomain).
		if prev != nil {
			return prev(ctx, ireq)
		}
		return nil, nil
	}
}

// OnResponse registers a Go callback for response interception.
func (p *Page) OnResponse(handler ResponseHandler) {
	prev := p.requestInterceptor
	p.requestInterceptor = func(ctx context.Context, ireq InterceptedRequest) (*InterceptResult, error) {
		if ireq.Stage == StageResponse {
			resp := &Response{
				URL:        ireq.URL,
				Method:     ireq.Method,
				StatusCode: ireq.StatusCode,
				Headers:    ireq.ResponseHeaders,
				Body:       ireq.Body,
				RequestID:  ireq.RequestID,
			}
			handler(resp)
			if resp.decided {
				return &resp.decision, nil
			}
		}
		if prev != nil {
			return prev(ctx, ireq)
		}
		return nil, nil
	}
}
