package network

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func TestDecompressResponseGzip(t *testing.T) {
	body := "<html><body>gzip compressed</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write([]byte(body))
		gz.Close()
		w.Write(buf.Bytes())
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := DecompressResponse(resp); err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestDecompressResponseBrotli(t *testing.T) {
	body := "<html><body>brotli compressed</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "br")
		var buf bytes.Buffer
		bw := brotli.NewWriter(&buf)
		bw.Write([]byte(body))
		bw.Close()
		w.Write(buf.Bytes())
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := DecompressResponse(resp); err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestDecompressResponseDeflate(t *testing.T) {
	body := "<html><body>deflate compressed</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "deflate")
		var buf bytes.Buffer
		fw, _ := flate.NewWriter(&buf, flate.DefaultCompression)
		fw.Write([]byte(body))
		fw.Close()
		w.Write(buf.Bytes())
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := DecompressResponse(resp); err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestDecompressResponseNoEncoding(t *testing.T) {
	body := "<html><body>plain text</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := DecompressResponse(resp); err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestDecodeResponseUTF8(t *testing.T) {
	body := "<html><body>hello</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestDecodeResponseShiftJIS(t *testing.T) {
	original := "こんにちは世界"
	encoded, _, _ := transform.Bytes(japanese.ShiftJIS.NewEncoder(), []byte(
		"<html><body>"+original+"</body></html>",
	))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=Shift_JIS")
		w.Write(encoded)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	if want := "<html><body>" + original + "</body></html>"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeResponseEUCJP(t *testing.T) {
	original := "日本語テスト"
	encoded, _, _ := transform.Bytes(japanese.EUCJP.NewEncoder(), []byte(
		"<html><body>"+original+"</body></html>",
	))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=EUC-JP")
		w.Write(encoded)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	if want := "<html><body>" + original + "</body></html>"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeResponseISO8859(t *testing.T) {
	original := "café"
	encoded, _, _ := transform.Bytes(charmap.ISO8859_1.NewEncoder(), []byte(
		"<html><body>"+original+"</body></html>",
	))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
		w.Write(encoded)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	if want := "<html><body>" + original + "</body></html>"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeResponseMetaCharset(t *testing.T) {
	original := "メタタグテスト"
	htmlContent := `<html><head><meta charset="Shift_JIS"></head><body>` + original + `</body></html>`
	encoded, _, _ := transform.Bytes(japanese.ShiftJIS.NewEncoder(), []byte(htmlContent))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No charset in Content-Type header
		w.Header().Set("Content-Type", "text/html")
		w.Write(encoded)
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	if want := htmlContent; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeResponseNoCharset(t *testing.T) {
	body := "<html><body>plain ascii</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	f := NewFetcher(nil)
	resp, err := f.Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(reader)
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}
