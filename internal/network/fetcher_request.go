package network

import (
	"context"
	"fmt"
	"net/http"
)

// FetchRequest performs an HTTP request with an arbitrary method and body,
// sharing the Fetcher's cookie jar, User-Agent, and TLS configuration.
//
// Default headers are subresource-style (Sec-Fetch-Dest: empty,
// Sec-Fetch-Mode: cors) rather than the navigate-style defaults used by
// Fetch/FetchContext, matching requests initiated by JS fetch()/XHR rather
// than top-level document navigation. extraHeaders are applied last and
// override these defaults.
//
// Because non-GET/HEAD methods may not be idempotent, the retry-on-transient-
// error/503/429 behavior of FetchContextWithHeaders only applies when method
// is GET or HEAD; other methods are attempted exactly once.
//
// The caller must close the response body.
func (f *Fetcher) FetchRequest(ctx context.Context, method, url string, extraHeaders http.Header, body []byte) (*http.Response, error) {
	merged := mergeHeaders(subresourceDefaultHeaders(), extraHeaders)

	if method == http.MethodGet || method == http.MethodHead {
		return f.fetchWithRetry(ctx, method, url, merged, body)
	}

	resp, err := f.doFetchMethod(ctx, method, url, merged, body)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	return resp, nil
}

// subresourceDefaultHeaders returns the Sec-Fetch-* header overrides for
// subresource-style requests (as opposed to top-level navigation).
func subresourceDefaultHeaders() http.Header {
	h := http.Header{}
	h.Set("Sec-Fetch-Dest", "empty")
	h.Set("Sec-Fetch-Mode", "cors")
	return h
}

// mergeHeaders returns a new http.Header containing base's entries with
// override's entries applied on top (override wins on key collisions).
func mergeHeaders(base, override http.Header) http.Header {
	merged := make(http.Header, len(base)+len(override))
	for k, v := range base {
		merged[k] = append([]string(nil), v...)
	}
	for k, v := range override {
		merged[k] = append([]string(nil), v...)
	}
	return merged
}
