package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRobotsAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Write([]byte("User-agent: *\nDisallow: /secret\n"))
		default:
			w.Write([]byte("<html><body>ok</body></html>"))
		}
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{ObeyRobots: true})

	// Allowed path
	doc, err := f.LoadDocument(srv.URL + "/public")
	if err != nil {
		t.Fatalf("allowed path failed: %v", err)
	}
	if doc == nil {
		t.Fatal("doc is nil for allowed path")
	}

	// Disallowed path
	_, err = f.LoadDocument(srv.URL + "/secret")
	if err == nil {
		t.Fatal("expected error for disallowed path, got nil")
	}

	// Subpath of disallowed
	_, err = f.LoadDocument(srv.URL + "/secret/page")
	if err == nil {
		t.Fatal("expected error for disallowed subpath, got nil")
	}
}

func TestRobotsSpecificAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Write([]byte("User-agent: Uzura\nDisallow: /blocked\n\nUser-agent: *\nAllow: /\n"))
		default:
			w.Write([]byte("<html><body>ok</body></html>"))
		}
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{ObeyRobots: true})

	_, err := f.LoadDocument(srv.URL + "/blocked")
	if err == nil {
		t.Fatal("expected error for agent-specific disallow, got nil")
	}
}

func TestRobotsDisabledByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Write([]byte("User-agent: *\nDisallow: /\n"))
		default:
			w.Write([]byte("<html><body>ok</body></html>"))
		}
	}))
	defer srv.Close()

	f := NewFetcher(nil) // ObeyRobots defaults to false
	doc, err := f.LoadDocument(srv.URL + "/page")
	if err != nil {
		t.Fatalf("should succeed when robots disabled: %v", err)
	}
	if doc == nil {
		t.Fatal("doc is nil")
	}
}

func TestRobotsMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()

	f := NewFetcher(&FetcherOptions{ObeyRobots: true})
	doc, err := f.LoadDocument(srv.URL + "/page")
	if err != nil {
		t.Fatalf("should succeed when robots.txt missing: %v", err)
	}
	if doc == nil {
		t.Fatal("doc is nil")
	}
}
