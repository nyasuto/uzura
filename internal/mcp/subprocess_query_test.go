package mcp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newQueryTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/landmarks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Landmarks</title></head>
<body>
<header><nav><a href="/">Home</a> | <a href="/about">About</a></nav></header>
<main>
<h1>Main Content</h1>
<p>Body text here.</p>
<form action="/search"><input type="text" name="q" placeholder="Search"><button type="submit">Go</button></form>
</main>
<aside><h2>Sidebar</h2><p>Related links</p></aside>
<footer><p>Footer info</p></footer>
</body></html>`)
	})

	mux.HandleFunc("/many-links", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Links</title></head><body>`)
		for i := 0; i < 150; i++ {
			fmt.Fprintf(w, `<a href="/page/%d">Link %d</a>`, i, i)
		}
		fmt.Fprint(w, `</body></html>`)
	})

	mux.HandleFunc("/huge-dom", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Huge</title></head><body><main>`)
		for i := 0; i < 1200; i++ {
			fmt.Fprintf(w, `<div class="item"><span>Item %d</span></div>`, i)
		}
		fmt.Fprint(w, `</main></body></html>`)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestSubprocess_SemanticTree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newQueryTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "semantic_tree", map[string]any{
		"url": ts.URL + "/landmarks",
	})
	text := result.Text()

	// Should detect landmarks.
	for _, keyword := range []string{"nav", "main", "footer"} {
		if !strings.Contains(strings.ToLower(text), keyword) {
			t.Errorf("expected '%s' in semantic_tree output, got: %s", keyword, truncate(text, 300))
		}
	}

	// Should detect interactive elements.
	if !strings.Contains(strings.ToLower(text), "input") && !strings.Contains(strings.ToLower(text), "search") {
		t.Errorf("expected interactive elements in semantic_tree output, got: %s", truncate(text, 300))
	}
}

func TestSubprocess_QueryH1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newQueryTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "query", map[string]any{
		"url":      ts.URL + "/landmarks",
		"selector": "h1",
	})
	text := result.Text()

	var resp queryResponseJSON
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("unmarshal query response: %v (raw: %s)", err, truncate(text, 200))
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 h1, got total=%d", resp.Total)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if !strings.Contains(resp.Results[0].Text, "Main Content") {
		t.Errorf("expected 'Main Content' in h1 text, got: %s", resp.Results[0].Text)
	}
}

func TestSubprocess_QueryLinks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newQueryTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "query", map[string]any{
		"url":      ts.URL + "/landmarks",
		"selector": "a",
	})
	text := result.Text()

	var resp queryResponseJSON
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total < 2 {
		t.Errorf("expected at least 2 links, got total=%d", resp.Total)
	}
}

func TestSubprocess_QueryAttribute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newQueryTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	result := p.callTool(t, "query", map[string]any{
		"url":       ts.URL + "/landmarks",
		"selector":  "a",
		"attribute": "href",
	})
	text := result.Text()

	var resp queryResponseJSON
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, r := range resp.Results {
		if r.Value == "" {
			t.Error("expected href attribute value, got empty")
		}
	}
}

func TestSubprocess_QueryPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newQueryTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	// First page: default limit=100.
	result1 := p.callTool(t, "query", map[string]any{
		"url":      ts.URL + "/many-links",
		"selector": "a",
	})
	var resp1 queryResponseJSON
	if err := json.Unmarshal([]byte(result1.Text()), &resp1); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp1.Total != 150 {
		t.Errorf("expected total=150, got %d", resp1.Total)
	}
	if resp1.Returned != 100 {
		t.Errorf("expected returned=100, got %d", resp1.Returned)
	}

	// Second page: offset=100.
	result2 := p.callTool(t, "query", map[string]any{
		"url":      ts.URL + "/many-links",
		"selector": "a",
		"offset":   100,
	})
	var resp2 queryResponseJSON
	if err := json.Unmarshal([]byte(result2.Text()), &resp2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp2.Total != 150 {
		t.Errorf("expected total=150, got %d", resp2.Total)
	}
	if resp2.Returned != 50 {
		t.Errorf("expected returned=50, got %d", resp2.Returned)
	}
	if resp2.Offset != 100 {
		t.Errorf("expected offset=100, got %d", resp2.Offset)
	}
}

func TestSubprocess_HugeDOMStability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	ts := newQueryTestServer(t)
	p := startMCP(t)
	p.initialize(t)

	// semantic_tree on huge DOM should not crash.
	result := p.callTool(t, "semantic_tree", map[string]any{
		"url": ts.URL + "/huge-dom",
	})
	text := result.Text()
	if text == "" {
		t.Error("expected non-empty semantic_tree output for huge DOM")
	}

	// query on huge DOM should return paginated results.
	qResult := p.callTool(t, "query", map[string]any{
		"url":      ts.URL + "/huge-dom",
		"selector": ".item",
	})
	var resp queryResponseJSON
	if err := json.Unmarshal([]byte(qResult.Text()), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1200 {
		t.Errorf("expected total=1200, got %d", resp.Total)
	}
	if resp.Returned != 100 {
		t.Errorf("expected returned=100 (default limit), got %d", resp.Returned)
	}
}

// queryResponseJSON mirrors the query tool's JSON response for test parsing.
type queryResponseJSON struct {
	Total    int `json:"total"`
	Returned int `json:"returned"`
	Offset   int `json:"offset"`
	Results  []struct {
		Text      string `json:"text"`
		Value     string `json:"value,omitempty"`
		OuterHTML string `json:"outerHTML"`
	} `json:"results"`
}
