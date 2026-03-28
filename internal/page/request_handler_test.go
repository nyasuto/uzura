package page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

func TestOnRequestContinue(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()

	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})
	p.OnRequest(func(req *page.Request) {
		h := make(http.Header)
		h.Set("Authorization", "Bearer secret")
		req.Continue(page.WithHeaders(h))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer secret")
	}
}

func TestOnRequestAbort(t *testing.T) {
	var serverHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHit = true
		w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()

	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})
	p.OnRequest(func(req *page.Request) {
		req.Abort("BlockedByClient")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.Navigate(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error from aborted request")
	}
	if !strings.Contains(err.Error(), "BlockedByClient") {
		t.Errorf("error = %v, want to contain BlockedByClient", err)
	}
	if serverHit {
		t.Error("server should not have been hit")
	}
}

func TestOnRequestFulfill(t *testing.T) {
	var serverHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHit = true
		w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()

	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})
	p.OnRequest(func(req *page.Request) {
		req.Fulfill(page.FulfillOption{
			Status:  200,
			Headers: map[string]string{"Content-Type": "text/html"},
			Body:    []byte("<html><body>mocked</body></html>"),
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if serverHit {
		t.Error("server should not have been hit with fulfill")
	}
	if p.Document() == nil {
		t.Fatal("document is nil")
	}
	body := p.Document().Body()
	if body == nil || body.TextContent() != "mocked" {
		t.Errorf("body text = %q, want %q", body.TextContent(), "mocked")
	}
}

func TestOnRequestURLRewrite(t *testing.T) {
	var lastPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()

	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})
	p.OnRequest(func(req *page.Request) {
		req.Continue(page.WithURL(srv.URL + "/redirected"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Navigate(ctx, srv.URL+"/original"); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if lastPath != "/redirected" {
		t.Errorf("path = %q, want /redirected", lastPath)
	}
}

func TestOnResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("X-Original", "yes")
		w.Write([]byte("<html><body>original</body></html>"))
	}))
	defer srv.Close()

	var capturedBody string
	var capturedStatus int
	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})
	p.OnResponse(func(resp *page.Response) {
		capturedBody = string(resp.Body)
		capturedStatus = resp.StatusCode
		resp.Continue()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if capturedStatus != 200 {
		t.Errorf("status = %d, want 200", capturedStatus)
	}
	if !strings.Contains(capturedBody, "original") {
		t.Errorf("body = %q, should contain 'original'", capturedBody)
	}
}

func TestOnResponseFulfill(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>server response</body></html>"))
	}))
	defer srv.Close()

	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})
	p.OnResponse(func(resp *page.Response) {
		resp.Fulfill(page.FulfillOption{
			Status:  200,
			Headers: map[string]string{"Content-Type": "text/html"},
			Body:    []byte("<html><body>replaced</body></html>"),
		})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	body := p.Document().Body()
	if body == nil || body.TextContent() != "replaced" {
		t.Errorf("body text = %q, want %q", body.TextContent(), "replaced")
	}
}

func TestOnRequestAndOnResponseChained(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>chained</body></html>"))
	}))
	defer srv.Close()

	var responseBodySeen string
	p := page.New(&page.Options{Fetcher: network.NewFetcher(nil)})

	// Register OnRequest first, then OnResponse.
	// Both should work on their respective stages.
	p.OnRequest(func(req *page.Request) {
		h := make(http.Header)
		h.Set("X-Custom", "injected")
		req.Continue(page.WithHeaders(h))
	})
	p.OnResponse(func(resp *page.Response) {
		responseBodySeen = string(resp.Body)
		resp.Continue()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Navigate(ctx, srv.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	if gotHeader != "injected" {
		t.Errorf("X-Custom = %q, want %q", gotHeader, "injected")
	}
	if !strings.Contains(responseBodySeen, "chained") {
		t.Errorf("response body = %q, should contain 'chained'", responseBodySeen)
	}
}
