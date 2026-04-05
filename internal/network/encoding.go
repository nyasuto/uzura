package network

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"github.com/andybalholm/brotli"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

var metaCharsetRe = regexp.MustCompile(`(?i)<meta[^>]+charset=["']?([^\s"';>]+)`)

// DecompressResponse replaces resp.Body with a decompressed reader based on
// the Content-Encoding header (gzip, br, deflate). If no content encoding
// is present, the body is left unchanged.
func DecompressResponse(resp *http.Response) error {
	ce := strings.TrimSpace(strings.ToLower(resp.Header.Get("Content-Encoding")))
	if ce == "" || ce == "identity" {
		return nil
	}

	var reader io.Reader
	switch ce {
	case "gzip":
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("gzip decompression: %w", err)
		}
		reader = gr
	case "br":
		reader = brotli.NewReader(resp.Body)
	case "deflate":
		reader = flate.NewReader(resp.Body)
	default:
		return nil
	}

	resp.Body = io.NopCloser(reader)
	resp.Header.Del("Content-Encoding")
	return nil
}

// DecodeResponse returns an io.Reader that decodes the response body to UTF-8.
// It detects the charset from the Content-Type header first, then falls back
// to scanning for a <meta charset> tag in the first 1024 bytes of the body.
// If no charset is found or the charset is already UTF-8, the body is returned as-is.
func DecodeResponse(resp *http.Response) (io.Reader, error) {
	// Try Content-Type header first
	charset := charsetFromContentType(resp.Header.Get("Content-Type"))

	if charset == "" {
		// Peek at the body to find <meta charset>
		peek := make([]byte, 1024)
		n, err := io.ReadFull(resp.Body, peek)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return nil, fmt.Errorf("reading body for charset detection: %w", err)
		}
		peek = peek[:n]

		charset = charsetFromMeta(peek)

		// Reconstruct reader with peeked bytes
		resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(peek), resp.Body))
	}

	if charset == "" || isUTF8(charset) {
		return resp.Body, nil
	}

	enc, err := htmlindex.Get(charset)
	if err != nil {
		// Unknown encoding; return body as-is
		return resp.Body, nil
	}

	return transform.NewReader(resp.Body, enc.NewDecoder()), nil
}

// charsetFromContentType extracts charset from a Content-Type header value.
func charsetFromContentType(ct string) string {
	if ct == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return params["charset"]
}

// charsetFromMeta scans HTML bytes for a <meta charset="..."> tag.
func charsetFromMeta(data []byte) string {
	m := metaCharsetRe.FindSubmatch(data)
	if m == nil {
		return ""
	}
	return string(m[1])
}

// isUTF8 returns true if the charset name refers to UTF-8.
func isUTF8(charset string) bool {
	enc, err := htmlindex.Get(charset)
	if err != nil {
		return false
	}
	return enc == encoding.Nop
}
