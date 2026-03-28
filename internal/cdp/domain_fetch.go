package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
}

// RequestPattern is a CDP Fetch.RequestPattern.
type RequestPattern struct {
	URLPattern   string `json:"urlPattern"`
	RequestStage string `json:"requestStage"` // "Request" or "Response"
}

type interceptDecision struct {
	action      page.InterceptAction
	errorReason string
}

// NewFetchDomain creates a FetchDomain.
func NewFetchDomain(p *page.Page) *FetchDomain {
	return &FetchDomain{
		page:    p,
		pending: make(map[string]chan interceptDecision),
	}
}

// SetPage sets the page after construction (for circular dependency wiring).
func (d *FetchDomain) SetPage(p *page.Page) {
	d.page = p
}

// Register adds Fetch domain handlers to the server.
func (d *FetchDomain) Register(s *Server) {
	s.HandleSession("Fetch.enable", d.enable)
	s.HandleSession("Fetch.disable", d.disable)
	s.HandleSession("Fetch.continueRequest", d.continueRequest)
	s.HandleSession("Fetch.failRequest", d.failRequest)
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
		RequestID string `json:"requestId"`
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

	ch <- interceptDecision{action: page.InterceptContinue}

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
		return nil, nil // not enabled, pass through
	}

	if !d.matchesAny(req.URL) {
		d.mu.Unlock()
		return nil, nil // no pattern matched, pass through
	}

	sess := d.session
	ch := make(chan interceptDecision, 1)
	d.pending[req.RequestID] = ch
	d.mu.Unlock()

	// Send Fetch.requestPaused event to the CDP client.
	if sess != nil {
		_ = sess.SendEvent("Fetch.requestPaused", map[string]interface{}{
			"requestId": req.RequestID,
			"request": map[string]interface{}{
				"url":     req.URL,
				"method":  req.Method,
				"headers": flattenHeaders(req.Headers),
			},
			"frameId":      "main",
			"resourceType": "Document",
			"networkId":    req.RequestID,
		})
	}

	// Wait for the client decision or context cancellation.
	select {
	case decision := <-ch:
		if decision.action == page.InterceptFail {
			return &page.InterceptResult{
				Action:      page.InterceptFail,
				ErrorReason: decision.errorReason,
			}, nil
		}
		return nil, nil // continue
	case <-ctx.Done():
		d.mu.Lock()
		delete(d.pending, req.RequestID)
		d.mu.Unlock()
		return nil, ctx.Err()
	}
}

// matchesAny checks if the URL matches any of the configured patterns.
// Must be called with d.mu held.
func (d *FetchDomain) matchesAny(url string) bool {
	for _, p := range d.patterns {
		if matchURLPattern(p.URLPattern, url) {
			return true
		}
	}
	return false
}

// matchURLPattern implements simple CDP URL pattern matching.
// "*" matches everything. Other patterns support leading/trailing wildcards.
func matchURLPattern(pattern, url string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}

	// Simple glob: only support * at the start and/or end.
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(url, pattern[1:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(url, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(url, pattern[:len(pattern)-1])
	}

	return url == pattern
}

func flattenHeaders(h map[string][]string) map[string]string {
	if h == nil {
		return nil
	}
	flat := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			flat[k] = v[0]
		}
	}
	return flat
}

