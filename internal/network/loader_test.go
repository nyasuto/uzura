package network

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func TestLoadDocumentBasic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><head><title>Test</title></head><body><p id="hello">world</p></body></html>`))
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	doc, err := f.LoadDocument(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if title := doc.Title(); title != "Test" {
		t.Errorf("title = %q, want %q", title, "Test")
	}

	elem := doc.GetElementById("hello")
	if elem == nil {
		t.Fatal("GetElementById returned nil")
	}
	if tc := elem.TextContent(); tc != "world" {
		t.Errorf("text = %q, want %q", tc, "world")
	}
}

func TestLoadDocumentShiftJIS(t *testing.T) {
	original := "日本語ページ"
	htmlContent := `<html><head><title>` + original + `</title></head><body></body></html>`
	encoded, _, _ := transform.Bytes(japanese.ShiftJIS.NewEncoder(), []byte(htmlContent))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
		w.Write(encoded)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	doc, err := f.LoadDocument(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if title := doc.Title(); title != original {
		t.Errorf("title = %q, want %q", title, original)
	}
}

func TestLoadDocumentNetworkError(t *testing.T) {
	f := NewFetcher(nil)
	_, err := f.LoadDocument("http://127.0.0.1:1") // unlikely to have a server
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

func TestLoadDocumentMalformedHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p>unclosed<div>tags<span>everywhere`))
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	doc, err := f.LoadDocument(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Parser should handle malformed HTML gracefully
	body := doc.Body()
	if body == nil {
		t.Fatal("body is nil")
	}
}
