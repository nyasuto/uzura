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

// Register adds Page domain handlers to the server.
func (d *PageDomain) Register(s *Server) {
	s.HandleSession("Page.enable", d.enable)
	s.HandleSession("Page.navigate", d.navigate)
	s.HandleSession("Page.getFrameTree", d.getFrameTree)
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
	tsData, _ := json.Marshal(map[string]interface{}{"timestamp": ts})

	events := []Event{
		{Method: "Page.domContentEventFired", Params: tsData},
		{Method: "Page.loadEventFired", Params: tsData},
	}

	r, merr := json.Marshal(map[string]interface{}{
		"frameId": frameID,
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
