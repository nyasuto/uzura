package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nyasuto/uzura/internal/page"
)

// PageDomain handles CDP Page domain methods.
type PageDomain struct {
	page *page.Page
}

// NewPageDomain creates a PageDomain wrapping the given page.
func NewPageDomain(p *page.Page) *PageDomain {
	return &PageDomain{page: p}
}

// Register adds Page domain handlers to the registry.
func (d *PageDomain) Register(r HandlerRegistry) {
	r.HandleSession("Page.enable", d.enable)
	r.HandleSession("Page.navigate", d.navigate)
	r.HandleSession("Page.getFrameTree", d.getFrameTree)
}

func (d *PageDomain) enable(_ *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	r, err := json.Marshal(struct{}{})
	return r, nil, err
}

func (d *PageDomain) navigate(sess *Session, params json.RawMessage) (json.RawMessage, []Event, error) {
	var p struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.URL == "" {
		return nil, nil, fmt.Errorf("url is required")
	}

	ctx, cancel := context.WithTimeout(sess.ctx, 30*time.Second)
	defer cancel()

	const frameID = "main"
	err := d.page.Navigate(ctx, p.URL)

	if err != nil {
		r, merr := json.Marshal(map[string]interface{}{
			"frameId":   frameID,
			"errorText": err.Error(),
		})
		return r, nil, merr
	}

	// Navigation succeeded — build lifecycle events to send after response.
	ts := float64(time.Now().UnixMilli()) / 1000.0

	url := d.page.URL()
	title := ""
	if doc := d.page.Document(); doc != nil {
		title = doc.Title()
	}

	initData, _ := json.Marshal(map[string]interface{}{
		"frameId":   frameID,
		"loaderId":  "loader-1",
		"name":      "init",
		"timestamp": ts,
	})
	ctxCreatedData, _ := json.Marshal(map[string]interface{}{
		"context": map[string]interface{}{
			"id":     1,
			"origin": url,
			"name":   "",
			"auxData": map[string]interface{}{
				"isDefault": true,
				"type":      "default",
				"frameId":   frameID,
			},
		},
	})

	frameNavigatedData, _ := json.Marshal(map[string]interface{}{
		"frame": map[string]interface{}{
			"id":             frameID,
			"loaderId":       "loader-1",
			"url":            url,
			"securityOrigin": url,
			"mimeType":       "text/html",
			"name":           title,
		},
		"type": "Navigation",
	})
	dclData, _ := json.Marshal(map[string]interface{}{
		"frameId":   frameID,
		"loaderId":  "loader-1",
		"name":      "DOMContentLoaded",
		"timestamp": ts,
	})
	loadData, _ := json.Marshal(map[string]interface{}{
		"frameId":   frameID,
		"loaderId":  "loader-1",
		"name":      "load",
		"timestamp": ts,
	})
	tsData, _ := json.Marshal(map[string]interface{}{"timestamp": ts})
	stoppedData, _ := json.Marshal(map[string]interface{}{"frameId": frameID})

	events := []Event{
		{Method: "Page.lifecycleEvent", Params: initData},
		{Method: "Page.frameNavigated", Params: frameNavigatedData},
		{Method: "Runtime.executionContextCreated", Params: ctxCreatedData},
		{Method: "Page.lifecycleEvent", Params: dclData},
		{Method: "Page.domContentEventFired", Params: tsData},
		{Method: "Page.lifecycleEvent", Params: loadData},
		{Method: "Page.loadEventFired", Params: tsData},
		{Method: "Page.frameStoppedLoading", Params: stoppedData},
	}

	r, merr := json.Marshal(map[string]interface{}{
		"frameId":  frameID,
		"loaderId": "loader-1",
	})
	return r, events, merr
}

func (d *PageDomain) getFrameTree(_ *Session, _ json.RawMessage) (json.RawMessage, []Event, error) {
	url := d.page.URL()
	if url == "" {
		url = "about:blank"
	}

	title := ""
	if doc := d.page.Document(); doc != nil {
		title = doc.Title()
	}

	r, err := json.Marshal(map[string]interface{}{
		"frameTree": map[string]interface{}{
			"frame": map[string]interface{}{
				"id":             "main",
				"url":            url,
				"loaderId":       "loader-1",
				"securityOrigin": url,
				"mimeType":       "text/html",
				"name":           title,
			},
		},
	})
	return r, nil, err
}
