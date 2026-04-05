package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func TestFetchFavicon(t *testing.T) {
	var mu sync.Mutex
	var faviconRequested bool
	var faviconReferer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/favicon.ico" {
			faviconRequested = true
			faviconReferer = r.Header.Get("Referer")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{EnableCookies: true})
	f.FetchFavicon(context.Background(), srv.URL+"/some/page")

	mu.Lock()
	defer mu.Unlock()
	if !faviconRequested {
		t.Error("favicon.ico was not requested")
	}
	if faviconReferer != srv.URL+"/some/page" {
		t.Errorf("Referer = %q, want %q", faviconReferer, srv.URL+"/some/page")
	}
}

func TestFetchFaviconHeaders(t *testing.T) {
	var headers http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			headers = r.Header.Clone()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	f.FetchFavicon(context.Background(), srv.URL+"/page")

	if headers == nil {
		t.Fatal("no favicon request was made")
	}

	tests := []struct {
		header string
		want   string
	}{
		{"Sec-Fetch-Dest", "image"},
		{"Sec-Fetch-Mode", "no-cors"},
		{"Sec-Fetch-Site", "same-origin"},
		{"Connection", "keep-alive"},
	}
	for _, tt := range tests {
		if got := headers.Get(tt.header); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestFetchResourceHints(t *testing.T) {
	var mu sync.Mutex
	requestedPaths := make(map[string]bool)
	requestMethods := make(map[string]string)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPaths[r.URL.Path] = true
		requestMethods[r.URL.Path] = r.Method
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resourceURLs := []string{
		srv.URL + "/style.css",
		srv.URL + "/app.js",
	}
	f.FetchResourceHints(context.Background(), srv.URL+"/page", resourceURLs)

	mu.Lock()
	defer mu.Unlock()

	for _, path := range []string{"/style.css", "/app.js"} {
		if !requestedPaths[path] {
			t.Errorf("resource %q was not requested", path)
		}
		if method := requestMethods[path]; method != http.MethodHead {
			t.Errorf("resource %q method = %q, want HEAD", path, method)
		}
	}
}

func TestFetchResourceHintsLimit(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	// Create more URLs than maxResourceHints (5).
	urls := make([]string, 10)
	for i := range urls {
		urls[i] = srv.URL + "/resource" + string(rune('0'+i)) + ".css"
	}
	f.FetchResourceHints(context.Background(), srv.URL+"/page", urls)

	mu.Lock()
	defer mu.Unlock()
	if requestCount > maxResourceHints {
		t.Errorf("request count = %d, want <= %d", requestCount, maxResourceHints)
	}
}

func TestFetchResourceHintsRelativeURLs(t *testing.T) {
	var mu sync.Mutex
	requestedPaths := make(map[string]bool)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPaths[r.URL.Path] = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resourceURLs := []string{
		"/css/style.css",
		"script.js",
	}
	f.FetchResourceHints(context.Background(), srv.URL+"/page/index.html", resourceURLs)

	mu.Lock()
	defer mu.Unlock()

	if !requestedPaths["/css/style.css"] {
		t.Error("absolute-path URL /css/style.css was not resolved")
	}
	if !requestedPaths["/page/script.js"] {
		t.Error("relative URL script.js was not resolved to /page/script.js")
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		rawURL  string
		want    string
	}{
		{"absolute", "https://example.com/page", "https://cdn.example.com/style.css", "https://cdn.example.com/style.css"},
		{"absolute path", "https://example.com/page", "/style.css", "https://example.com/style.css"},
		{"relative", "https://example.com/dir/page", "style.css", "https://example.com/dir/style.css"},
		{"empty", "https://example.com/page", "", ""},
		{"spaces", "https://example.com/page", "  /style.css  ", "https://example.com/style.css"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := url.Parse(tt.baseURL)
			if err != nil {
				t.Fatalf("parse base URL: %v", err)
			}
			got := resolveURL(base, tt.rawURL)
			if got != tt.want {
				t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.baseURL, tt.rawURL, got, tt.want)
			}
		})
	}
}
