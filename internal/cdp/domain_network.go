package cdp

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nyasuto/uzura/internal/page"
)

// NetworkDomain handles CDP Network domain methods.
type NetworkDomain struct {
	page    *page.Page
	session *Session
	mu      sync.Mutex
	bodies  map[string][]byte // requestId → response body
}

// NewNetworkDomain creates a NetworkDomain wrapping the given page.
func NewNetworkDomain(p *page.Page) *NetworkDomain {
	return &NetworkDomain{
		page:   p,
		bodies: make(map[string][]byte),
	}
}

// SetPage sets the page after construction (for circular dependency wiring).
func (d *NetworkDomain) SetPage(p *page.Page) {
	d.page = p
}

// Register adds Network domain handlers to the server.
func (d *NetworkDomain) Register(s *Server) {
	s.HandleSession("Network.enable", d.enable)
	s.Handle("Network.getResponseBody", d.getResponseBody)
}

// Observer returns a NetworkObserver callback for use with page.Options.
func (d *NetworkDomain) Observer() page.NetworkObserver {
	return d.handleNetworkEvent
}

func (d *NetworkDomain) enable(sess *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	d.mu.Lock()
	d.session = sess
	d.mu.Unlock()

	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *NetworkDomain) getResponseBody(params json.RawMessage) (json.RawMessage, error) {
	var p struct {
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	d.mu.Lock()
	body, ok := d.bodies[p.RequestID]
	d.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no response body for request %s", p.RequestID)
	}

	r, err := json.Marshal(map[string]interface{}{
		"body":          string(body),
		"base64Encoded": false,
	})
	return r, err
}

func (d *NetworkDomain) handleNetworkEvent(evt page.NetworkEvent) {
	d.mu.Lock()
	sess := d.session
	d.mu.Unlock()

	if sess == nil {
		return
	}

	switch evt.Type {
	case page.NetworkRequestWillBeSent:
		d.sendRequestWillBeSent(sess, evt)
	case page.NetworkResponseReceived:
		d.sendResponseReceived(sess, evt)
	case page.NetworkLoadingFinished:
		d.storeBody(evt)
		d.sendLoadingFinished(sess, evt)
	case page.NetworkLoadingFailed:
		d.sendLoadingFailed(sess, evt)
	}
}

func (d *NetworkDomain) sendRequestWillBeSent(sess *Session, evt page.NetworkEvent) {
	headers := make(map[string]string)
	for k, v := range evt.Headers {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	_ = sess.SendEvent("Network.requestWillBeSent", map[string]interface{}{
		"requestId": evt.RequestID,
		"loaderId":  "loader-1",
		"documentURL": evt.URL,
		"timestamp": evt.Timestamp,
		"type":      "Document",
		"frameId":   "main",
		"request": map[string]interface{}{
			"url":     evt.URL,
			"method":  evt.Method,
			"headers": headers,
		},
	})
}

func (d *NetworkDomain) sendResponseReceived(sess *Session, evt page.NetworkEvent) {
	headers := make(map[string]string)
	for k, v := range evt.RespHeaders {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	_ = sess.SendEvent("Network.responseReceived", map[string]interface{}{
		"requestId": evt.RequestID,
		"loaderId":  "loader-1",
		"timestamp": evt.Timestamp,
		"type":      "Document",
		"frameId":   "main",
		"response": map[string]interface{}{
			"url":        evt.URL,
			"status":     evt.StatusCode,
			"statusText": evt.StatusText,
			"headers":    headers,
			"mimeType":   evt.MimeType,
		},
	})
}

func (d *NetworkDomain) sendLoadingFinished(sess *Session, evt page.NetworkEvent) {
	_ = sess.SendEvent("Network.loadingFinished", map[string]interface{}{
		"requestId":         evt.RequestID,
		"timestamp":         evt.Timestamp,
		"encodedDataLength": evt.EncodedDataLength,
	})
}

func (d *NetworkDomain) sendLoadingFailed(sess *Session, evt page.NetworkEvent) {
	_ = sess.SendEvent("Network.loadingFailed", map[string]interface{}{
		"requestId": evt.RequestID,
		"timestamp": evt.Timestamp,
		"type":      "Document",
		"errorText": evt.ErrorText,
	})
}

func (d *NetworkDomain) storeBody(evt page.NetworkEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bodies[evt.RequestID] = evt.Body
}
