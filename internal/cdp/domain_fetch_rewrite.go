package cdp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nyasuto/uzura/internal/page"
)

func (d *FetchDomain) fulfillRequest(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		RequestID       string        `json:"requestId"`
		ResponseCode    int           `json:"responseCode"`
		ResponseHeaders []headerEntry `json:"responseHeaders"`
		Body            string        `json:"body"` // base64-encoded
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

	var body []byte
	if p.Body != "" {
		var err error
		body, err = base64.StdEncoding.DecodeString(p.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid base64 body: %w", err)
		}
	}

	code := p.ResponseCode
	if code == 0 {
		code = 200
	}

	ch <- interceptDecision{
		action:          page.InterceptFulfill,
		responseCode:    code,
		responseHeaders: headerEntriesToMap(p.ResponseHeaders),
		responseBody:    body,
	}

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *FetchDomain) getResponseBody(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}

	d.mu.Lock()
	pr, ok := d.pausedResponses[p.RequestID]
	d.mu.Unlock()

	if !ok {
		return nil, nil, fmt.Errorf("no paused response with id %s", p.RequestID)
	}

	r, err := json.Marshal(map[string]interface{}{
		"body":          base64.StdEncoding.EncodeToString(pr.body),
		"base64Encoded": true,
	})
	return r, nil, err
}

func (d *FetchDomain) continueResponse(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		RequestID       string        `json:"requestId"`
		ResponseCode    int           `json:"responseCode"`
		ResponseHeaders []headerEntry `json:"responseHeaders"`
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

	decision := interceptDecision{action: page.InterceptContinue}
	if p.ResponseCode > 0 {
		decision.respStatusCode = p.ResponseCode
	}
	if len(p.ResponseHeaders) > 0 {
		decision.respHeaders = headerEntriesToMap(p.ResponseHeaders)
	}
	ch <- decision

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *FetchDomain) decisionToResult(dec interceptDecision) *page.InterceptResult {
	switch dec.action {
	case page.InterceptFail:
		return &page.InterceptResult{
			Action:      page.InterceptFail,
			ErrorReason: dec.errorReason,
		}
	case page.InterceptFulfill:
		return &page.InterceptResult{
			Action:          page.InterceptFulfill,
			ResponseCode:    dec.responseCode,
			ResponseHeaders: dec.responseHeaders,
			ResponseBody:    dec.responseBody,
		}
	default:
		if dec.url == "" && dec.headers == nil && dec.respStatusCode == 0 && dec.respHeaders == nil {
			return nil
		}
		result := &page.InterceptResult{Action: page.InterceptContinue}
		if dec.url != "" {
			result.URL = dec.url
		}
		if dec.headers != nil {
			h := make(http.Header)
			for k, v := range dec.headers {
				h.Set(k, v)
			}
			result.Headers = h
		}
		if dec.respStatusCode > 0 {
			result.RespStatusCode = dec.respStatusCode
		}
		if dec.respHeaders != nil {
			result.RespHeaders = dec.respHeaders
		}
		return result
	}
}

// matchesAnyStage checks if the URL matches any pattern for the given stage.
// Must be called with d.mu held.
func (d *FetchDomain) matchesAnyStage(url, stage string) bool {
	for _, p := range d.patterns {
		patternStage := p.RequestStage
		if patternStage == "" {
			patternStage = "Request"
		}
		if patternStage == stage && matchURLPattern(p.URLPattern, url) {
			return true
		}
	}
	return false
}

// headerEntry represents a single CDP header entry {name, value}.
type headerEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// headerEntriesToMap converts CDP header entries to a flat map.
func headerEntriesToMap(entries []headerEntry) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Name] = e.Value
	}
	return m
}

// headerToEntries converts http.Header to CDP header entry array.
func headerToEntries(h http.Header) []headerEntry {
	if h == nil {
		return nil
	}
	entries := make([]headerEntry, 0, len(h))
	for k, vals := range h {
		if len(vals) > 0 {
			entries = append(entries, headerEntry{Name: k, Value: vals[0]})
		}
	}
	return entries
}

// matchURLPattern implements simple CDP URL pattern matching.
// "*" matches everything. Other patterns support leading/trailing wildcards.
func matchURLPattern(pattern, url string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}
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
