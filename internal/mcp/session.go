package mcp

import (
	"context"
	"sync"

	"github.com/nyasuto/uzura/internal/page"
)

// PageSession manages a cache of pages keyed by URL.
// When the same URL is requested multiple times, the existing page's DOM is reused.
type PageSession struct {
	mu    sync.Mutex
	pages map[string]*page.Page
}

// NewPageSession creates a new page session cache.
func NewPageSession() *PageSession {
	return &PageSession{
		pages: make(map[string]*page.Page),
	}
}

// GetOrNavigate returns a cached page for the URL, or navigates to it and caches the result.
func (ps *PageSession) GetOrNavigate(ctx context.Context, url string) (*page.Page, error) {
	ps.mu.Lock()
	if p, ok := ps.pages[url]; ok {
		ps.mu.Unlock()
		return p, nil
	}
	ps.mu.Unlock()

	p := page.New(nil)
	if err := p.Navigate(ctx, url); err != nil {
		p.Close()
		return nil, err
	}

	ps.mu.Lock()
	// Check again in case another goroutine added the same URL.
	if existing, ok := ps.pages[url]; ok {
		ps.mu.Unlock()
		p.Close()
		return existing, nil
	}
	ps.pages[url] = p
	ps.mu.Unlock()
	return p, nil
}

// Close closes all cached pages and clears the cache.
func (ps *PageSession) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for _, p := range ps.pages {
		p.Close()
	}
	ps.pages = make(map[string]*page.Page)
}
