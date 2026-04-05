package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>hello</body></html>"))
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("content-type = %q, want %q", ct, "text/html; charset=utf-8")
	}
}

func TestFetchUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	resp.Body.Close()

	if gotUA != DefaultUserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, DefaultUserAgent)
	}

	// Custom User-Agent
	custom := "CustomBot/1.0"
	f2 := NewFetcher(&FetcherOptions{UserAgent: custom})
	resp2, err := f2.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	resp2.Body.Close()

	if gotUA != custom {
		t.Errorf("User-Agent = %q, want %q", gotUA, custom)
	}
}

func TestFetchRedirect(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/final" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("done"))
			return
		}
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL + "/start")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if hits != 2 {
		t.Errorf("hits = %d, want 2", hits)
	}
}

func TestFetchRedirectLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	_, err := f.Fetch(srv.URL + "/loop")
	if err == nil {
		t.Fatal("expected error for redirect loop, got nil")
	}
}

func TestFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{Timeout: 100 * time.Millisecond})
	_, err := f.Fetch(srv.URL)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestFetchDefaultTimeout(t *testing.T) {
	f := NewFetcher(nil)
	if f.client.Timeout != DefaultTimeout {
		t.Errorf("default timeout = %v, want %v", f.client.Timeout, DefaultTimeout)
	}
}

func TestFetchBrowserHeaders(t *testing.T) {
	var headers http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	resp.Body.Close()

	checks := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}
	for name, want := range checks {
		got := headers.Get(name)
		if got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
}

func TestFetch404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
