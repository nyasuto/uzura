package mcp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer creates an httptest.Server with test pages.
// The server is automatically closed when the test finishes.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/basic", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Test Page</title></head>
<body>
<h1>Hello Uzura</h1>
<p>This is a test page for subprocess testing.</p>
</body></html>`)
	})

	mux.HandleFunc("/with-script", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Script Page</title>
<script>console.log("should be removed");</script>
<style>body { color: red; }</style>
</head>
<body>
<h1>Content Only</h1>
<script>var x = 1;</script>
<p>Visible text here.</p>
</body></html>`)
	})

	mux.HandleFunc("/markdown", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Markdown Test</title></head>
<body>
<article>
<h1>Article Title</h1>
<p>First paragraph of the article.</p>
<h2>Section Two</h2>
<p>Second paragraph with <a href="/link">a link</a>.</p>
<ul>
<li>Item one</li>
<li>Item two</li>
</ul>
</article>
</body></html>`)
	})

	mux.HandleFunc("/large", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Large Page</title></head><body>`)
		// Generate >100KB of content.
		for i := 0; i < 5000; i++ {
			fmt.Fprintf(w, `<p>Paragraph %d with some content to bulk up the page size for testing truncation.</p>`, i)
		}
		fmt.Fprint(w, `</body></html>`)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestSubprocess_BrowseText(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "browse", map[string]any{
		"url":    ts.URL + "/basic",
		"format": "text",
	})
	text := result.Text()

	if !strings.Contains(text, "Hello Uzura") {
		t.Errorf("expected 'Hello Uzura' in text output, got: %s", truncate(text, 200))
	}
	if !strings.Contains(text, "test page for subprocess testing") {
		t.Errorf("expected body text in output, got: %s", truncate(text, 200))
	}
}

func TestSubprocess_BrowseMarkdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "browse", map[string]any{
		"url":    ts.URL + "/markdown",
		"format": "markdown",
	})
	text := result.Text()

	// Should contain markdown heading.
	if !strings.Contains(text, "Article Title") {
		t.Errorf("expected 'Article Title' in markdown output, got: %s", truncate(text, 300))
	}
	// Should contain link text.
	if !strings.Contains(text, "a link") {
		t.Errorf("expected link text in markdown output, got: %s", truncate(text, 300))
	}
}

func TestSubprocess_BrowseScriptStyleRemoval(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "browse", map[string]any{
		"url":    ts.URL + "/with-script",
		"format": "text",
	})
	text := result.Text()

	// Script content should be removed.
	if strings.Contains(text, "should be removed") {
		t.Error("script content should be removed from text output")
	}
	if strings.Contains(text, "var x = 1") {
		t.Error("inline script should be removed from text output")
	}
	// Body text should be present.
	if !strings.Contains(text, "Visible text here") {
		t.Errorf("expected visible text in output, got: %s", truncate(text, 200))
	}
	if !strings.Contains(text, "Content Only") {
		t.Errorf("expected heading in output, got: %s", truncate(text, 200))
	}
}

func TestSubprocess_BrowseLargePageTruncation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	// Default max_length is 100KB. The large page is >100KB.
	result := p.callTool(t, "browse", map[string]any{
		"url":    ts.URL + "/large",
		"format": "text",
	})
	text := result.Text()

	if !strings.Contains(text, "[truncated]") {
		t.Error("expected [truncated] marker in large page output")
	}
}

func TestSubprocess_BrowseMaxLength(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	// Use a small max_length to force truncation.
	result := p.callTool(t, "browse", map[string]any{
		"url":        ts.URL + "/basic",
		"format":     "text",
		"max_length": 50,
	})
	text := result.Text()

	if !strings.Contains(text, "[truncated]") {
		t.Errorf("expected [truncated] with max_length=50, got: %s", truncate(text, 100))
	}
}

// truncate shortens a string for error messages.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
