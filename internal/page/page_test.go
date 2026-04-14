package page

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nyasuto/uzura/internal/network"
)

func TestNavigateBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><head><title>Test Page</title></head><body><p id="msg">hello</p></body></html>`))
	}))
	defer srv.Close()

	p := New(nil)
	if err := p.Navigate(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}

	doc := p.Document()
	if doc == nil {
		t.Fatal("document is nil after Navigate")
	}
	if title := doc.Title(); title != "Test Page" {
		t.Errorf("title = %q, want %q", title, "Test Page")
	}
	if p.URL() != srv.URL {
		t.Errorf("URL = %q, want %q", p.URL(), srv.URL)
	}

	elem := doc.GetElementById("msg")
	if elem == nil {
		t.Fatal("GetElementById returned nil")
	}
	if tc := elem.TextContent(); tc != "hello" {
		t.Errorf("textContent = %q, want %q", tc, "hello")
	}
}

func TestNavigateQuerySelector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><body><ul><li class="item">A</li><li class="item">B</li></ul></body></html>`))
	}))
	defer srv.Close()

	p := New(nil)
	if err := p.Navigate(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}

	doc := p.Document()
	elems, err := doc.QuerySelectorAll(".item")
	if err != nil {
		t.Fatal(err)
	}
	if len(elems) != 2 {
		t.Errorf("querySelectorAll('.item') returned %d elements, want 2", len(elems))
	}
}

func TestNavigateContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte(`<html><body>slow</body></html>`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	p := New(nil)
	err := p.Navigate(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestNavigateNetworkError(t *testing.T) {
	p := New(nil)
	err := p.Navigate(context.Background(), "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

func TestNavigateMalformedHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p>unclosed<div>tags`))
	}))
	defer srv.Close()

	p := New(nil)
	if err := p.Navigate(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}

	doc := p.Document()
	if doc == nil {
		t.Fatal("document is nil")
	}
	if doc.Body() == nil {
		t.Fatal("body is nil")
	}
}

func TestNavigateWithCustomFetcher(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != "CustomBot/1.0" {
			t.Errorf("User-Agent = %q, want %q", ua, "CustomBot/1.0")
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Custom</title></head><body></body></html>`))
	}))
	defer srv.Close()

	f := network.NewFetcher(&network.FetcherOptions{
		UserAgent: "CustomBot/1.0",
	})
	p := New(&Options{Fetcher: f})
	if err := p.Navigate(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if title := p.Document().Title(); title != "Custom" {
		t.Errorf("title = %q, want %q", title, "Custom")
	}
}

func TestNavigateReplacesDocument(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		calls++
		w.Header().Set("Content-Type", "text/html")
		if calls == 1 {
			w.Write([]byte(`<html><head><title>First</title></head><body></body></html>`))
		} else {
			w.Write([]byte(`<html><head><title>Second</title></head><body></body></html>`))
		}
	}))
	defer srv.Close()

	p := New(nil)
	if err := p.Navigate(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if title := p.Document().Title(); title != "First" {
		t.Errorf("title = %q, want %q", title, "First")
	}

	if err := p.Navigate(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if title := p.Document().Title(); title != "Second" {
		t.Errorf("title = %q, want %q", title, "Second")
	}
}

func TestDocumentNilBeforeNavigate(t *testing.T) {
	p := New(nil)
	if p.Document() != nil {
		t.Error("document should be nil before Navigate")
	}
	if p.URL() != "" {
		t.Error("URL should be empty before Navigate")
	}
}
