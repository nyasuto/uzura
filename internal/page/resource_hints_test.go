package page

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/html"
)

func TestExtractResourceURLs(t *testing.T) {
	htmlStr := `<html><head>
		<link rel="stylesheet" href="/css/style.css">
		<link rel="icon" href="/favicon.png">
		<script src="/js/app.js"></script>
	</head><body>
		<script src="/js/vendor.js"></script>
		<p>hello</p>
	</body></html>`

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatal(err)
	}

	urls := extractResourceURLs(doc)

	expected := []string{"/css/style.css", "/js/app.js", "/js/vendor.js"}
	if len(urls) != len(expected) {
		t.Fatalf("got %d URLs, want %d: %v", len(urls), len(expected), urls)
	}
	for i, want := range expected {
		if urls[i] != want {
			t.Errorf("urls[%d] = %q, want %q", i, urls[i], want)
		}
	}
}

func TestExtractResourceURLsIgnoresNonStylesheetLinks(t *testing.T) {
	htmlStr := `<html><head>
		<link rel="icon" href="/favicon.png">
		<link rel="preconnect" href="https://cdn.example.com">
		<link rel="stylesheet" href="/style.css">
	</head><body></body></html>`

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatal(err)
	}

	urls := extractResourceURLs(doc)
	if len(urls) != 1 || urls[0] != "/style.css" {
		t.Errorf("got %v, want [/style.css]", urls)
	}
}

func TestExtractResourceURLsEmpty(t *testing.T) {
	doc := dom.NewDocument()
	urls := extractResourceURLs(doc)
	if len(urls) != 0 {
		t.Errorf("got %d URLs for empty doc, want 0", len(urls))
	}
}

func TestNavigateRefererHeader(t *testing.T) {
	var mu sync.Mutex
	requests := make(map[string]http.Header)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.URL.Path] = r.Header.Clone()
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>hello</body></html>`))
	}))
	defer srv.Close()

	p := New(nil)

	// First navigation: no Referer.
	if err := p.Navigate(context.Background(), srv.URL+"/page1"); err != nil {
		t.Fatal(err)
	}
	// Wait for background resource hints to complete.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	page1Headers := requests["/page1"]
	mu.Unlock()
	if referer := page1Headers.Get("Referer"); referer != "" {
		t.Errorf("first navigation should have no Referer, got %q", referer)
	}

	// Second navigation: Referer should be set to previous URL.
	if err := p.Navigate(context.Background(), srv.URL+"/page2"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	page2Headers := requests["/page2"]
	mu.Unlock()
	expectedReferer := srv.URL + "/page1"
	if referer := page2Headers.Get("Referer"); referer != expectedReferer {
		t.Errorf("Referer = %q, want %q", referer, expectedReferer)
	}
}

func TestNavigateConnectionKeepAlive(t *testing.T) {
	var connectionHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			connectionHeader = r.Header.Get("Connection")
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>hello</body></html>`))
	}))
	defer srv.Close()

	p := New(nil)
	if err := p.Navigate(context.Background(), srv.URL+"/"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	if connectionHeader != "keep-alive" {
		t.Errorf("Connection = %q, want %q", connectionHeader, "keep-alive")
	}
}

func TestNavigateFaviconAndResourceHints(t *testing.T) {
	var mu sync.Mutex
	requestedPaths := make(map[string]bool)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPaths[r.URL.Path] = true
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<link rel="stylesheet" href="/css/main.css">
			<script src="/js/app.js"></script>
		</head><body>hello</body></html>`))
	}))
	defer srv.Close()

	p := New(nil)
	if err := p.Navigate(context.Background(), srv.URL+"/page"); err != nil {
		t.Fatal(err)
	}

	// Wait for background resource hints.
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !requestedPaths["/favicon.ico"] {
		t.Error("favicon.ico was not requested after navigation")
	}
	if !requestedPaths["/css/main.css"] {
		t.Error("CSS resource /css/main.css was not requested")
	}
	if !requestedPaths["/js/app.js"] {
		t.Error("JS resource /js/app.js was not requested")
	}
}
