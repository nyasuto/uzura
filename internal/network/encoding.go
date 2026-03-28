package network

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

var metaCharsetRe = regexp.MustCompile(`(?i)<meta[^>]+charset=["']?([^\s"';>]+)`)

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
