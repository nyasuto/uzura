package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/nyasuto/uzura/internal/page"
)

// FetchDomain handles CDP Fetch domain methods for request interception.
type FetchDomain struct {
	page    *page.Page
	session *Session
	mu      sync.Mutex
	enabled bool

	// patterns stores the URL patterns to intercept.
	patterns []RequestPattern

	// pending maps requestId → channel for sending the intercept decision.
	pending map[string]chan interceptDecision

	// pausedResponses stores response data for requests paused at response stage.
	pausedResponses map[string]*pausedResponse
}

type pausedResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
}

// RequestPattern is a CDP Fetch.RequestPattern.
type RequestPattern struct {
	URLPattern   string `json:"urlPattern"`
	RequestStage string `json:"requestStage"` // "Request" or "Response"
}

type interceptDecision struct {
	action      page.InterceptAction
	errorReason string

	// continueRequest overrides.
	url     string
	headers map[string]string

	// fulfillRequest fields.
	responseCode    int
	responseHeaders map[string]string
	responseBody    []byte

	// continueResponse overrides.
	respStatusCode int
	respHeaders    map[string]string
}

// NewFetchDomain creates a FetchDomain.
func NewFetchDomain(p *page.Page) *FetchDomain {
	return &FetchDomain{
		page:            p,
		pending:         make(map[string]chan interceptDecision),
		pausedResponses: make(map[string]*pausedResponse),
	}
}

// SetPage sets the page after construction (for circular dependency wiring).
func (d *FetchDomain) SetPage(p *page.Page) {
	d.page = p
}

// Register adds Fetch domain handlers to the registry.
func (d *FetchDomain) Register(r HandlerRegistry) {
	r.HandleSession("Fetch.enable", d.enable)
	r.HandleSession("Fetch.disable", d.disable)
	r.HandleSession("Fetch.continueRequest", d.continueRequest)
	r.HandleSession("Fetch.failRequest", d.failRequest)
	r.HandleSession("Fetch.fulfillRequest", d.fulfillRequest)
	r.HandleSession("Fetch.getResponseBody", d.getResponseBody)
	r.HandleSession("Fetch.continueResponse", d.continueResponse)
}

// Interceptor returns a RequestInterceptor for use with page.Options.
func (d *FetchDomain) Interceptor() page.RequestInterceptor {
	return d.intercept
}

func (d *FetchDomain) enable(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		Patterns []RequestPattern `json:"patterns"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	d.mu.Lock()
	d.session = sess
	d.enabled = true
	d.patterns = p.Patterns
	// Default: intercept all requests if no patterns specified.
	if len(d.patterns) == 0 {
		d.patterns = []RequestPattern{{URLPattern: "*", RequestStage: "Request"}}
	}
	d.mu.Unlock()

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *FetchDomain) disable(sess *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	d.mu.Lock()
	d.enabled = false
	d.session = nil
	d.patterns = nil
	// Release any pending intercepted requests so they don't hang.
	for id, ch := range d.pending {
		ch <- interceptDecision{action: page.InterceptContinue}
		delete(d.pending, id)
	}
	d.mu.Unlock()

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *FetchDomain) continueRequest(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		RequestID string        `json:"requestId"`
		URL       string        `json:"url"`
		Headers   []headerEntry `json:"headers"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	d.mu.Lock()
	ch, ok := d.pending[p.RequestID]
	if ok {
		delete(d.pending, p.RequestID)
	}
	d.mu.Unlock()

	if !ok {
		return nil, nil, fmt.Errorf("no intercepted request with id %s", p.RequestID)
	}

	decision := interceptDecision{action: page.InterceptContinue, url: p.URL}
	if len(p.Headers) > 0 {
		decision.headers = headerEntriesToMap(p.Headers)
	}
	ch <- decision

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *FetchDomain) failRequest(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		RequestID string `json:"requestId"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	d.mu.Lock()
	ch, ok := d.pending[p.RequestID]
	if ok {
		delete(d.pending, p.RequestID)
	}
	d.mu.Unlock()

	if !ok {
		return nil, nil, fmt.Errorf("no intercepted request with id %s", p.RequestID)
	}

	reason := p.Reason
	if reason == "" {
		reason = "Failed"
	}
	ch <- interceptDecision{action: page.InterceptFail, errorReason: reason}

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

// intercept is the RequestInterceptor callback wired into the Page.
func (d *FetchDomain) intercept(ctx context.Context, req page.InterceptedRequest) (*page.InterceptResult, error) {
	d.mu.Lock()
	if !d.enabled {
		d.mu.Unlock()
		return nil, nil
	}

	stage := "Request"
	if req.Stage == page.StageResponse {
		stage = "Response"
	}
	if !d.matchesAnyStage(req.URL, stage) {
		d.mu.Unlock()
		return nil, nil
	}

	sess := d.session
	ch := make(chan interceptDecision, 1)
	d.pending[req.RequestID] = ch

	// Store response data for getResponseBody at response stage.
	if req.Stage == page.StageResponse {
		d.pausedResponses[req.RequestID] = &pausedResponse{
			statusCode: req.StatusCode,
			headers:    req.ResponseHeaders,
			body:       req.Body,
		}
	}
	d.mu.Unlock()

	// Build Fetch.requestPaused event payload.
	evt := map[string]interface{}{
		"requestId": req.RequestID,
		"request": map[string]interface{}{
			"url":     req.URL,
			"method":  req.Method,
			"headers": flattenHeaders(req.Headers),
		},
		"frameId":      "main",
		"resourceType": "Document",
		"networkId":    req.RequestID,
	}
	if req.Stage == page.StageResponse {
		evt["responseStatusCode"] = req.StatusCode
		evt["responseHeaders"] = headerToEntries(req.ResponseHeaders)
	}

	if sess != nil {
		_ = sess.SendEvent("Fetch.requestPaused", evt)
	}

	select {
	case decision := <-ch:
		d.mu.Lock()
		delete(d.pausedResponses, req.RequestID)
		d.mu.Unlock()
		return d.decisionToResult(decision), nil
	case <-ctx.Done():
		d.mu.Lock()
		delete(d.pending, req.RequestID)
		delete(d.pausedResponses, req.RequestID)
		d.mu.Unlock()
		return nil, ctx.Err()
	}
}
