package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPageSession_CachesPage(t *testing.T) {
	var fetchCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetchCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Cached</h1></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterBrowseTool(s)

	// First call fetches the page.
	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result1 := callTool(s, "browse", args)
	if result1.IsError {
		t.Fatalf("first call error: %s", result1.Content[0].Text)
	}

	// Second call should reuse the cached page.
	_, result2 := callTool(s, "browse", args)
	if result2.IsError {
		t.Fatalf("second call error: %s", result2.Content[0].Text)
	}

	if result1.Content[0].Text != result2.Content[0].Text {
		t.Error("cached results should be identical")
	}

	// The server should have been hit only once.
	if got := fetchCount.Load(); got != 1 {
		t.Errorf("fetch count = %d, want 1 (page should be cached)", got)
	}
}

func TestPageSession_DifferentURLsNotCached(t *testing.T) {
	var fetchCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount.Add(1)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>%s</body></html>`, r.URL.Path)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterBrowseTool(s)

	callTool(s, "browse", fmt.Sprintf(`{"url":%q}`, ts.URL+"/a"))
	callTool(s, "browse", fmt.Sprintf(`{"url":%q}`, ts.URL+"/b"))

	if got := fetchCount.Load(); got != 2 {
		t.Errorf("fetch count = %d, want 2 (different URLs should not cache)", got)
	}
}

func TestPageSession_Close(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>test</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterBrowseTool(s)

	callTool(s, "browse", fmt.Sprintf(`{"url":%q}`, ts.URL))
	s.Session.Close()

	// After close, pages map should be empty (new fetch on next call).
	if len(s.Session.pages) != 0 {
		t.Errorf("expected empty pages after Close, got %d", len(s.Session.pages))
	}
}
