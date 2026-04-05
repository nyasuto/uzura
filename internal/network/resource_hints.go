// Package network provides HTTP fetching for document loading.
package network

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ResourceHintTimeout is the timeout for background resource hint requests.
const ResourceHintTimeout = 5 * time.Second

// maxResourceHints limits the number of background resource requests per page.
const maxResourceHints = 5

// FetchFavicon sends a background GET request for /favicon.ico relative to the
// given page URL. The response body is discarded. This mimics browser behavior
// where a favicon request is always sent on initial page load.
func (f *Fetcher) FetchFavicon(ctx context.Context, pageURL string) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return
	}
	faviconURL := parsed.Scheme + "://" + parsed.Host + "/favicon.ico"

	ctx, cancel := context.WithTimeout(ctx, ResourceHintTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, faviconURL, http.NoBody)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Referer", pageURL)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "image")
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := f.client.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain body before close
	resp.Body.Close()
}

// FetchResourceHints sends background HEAD requests for the given resource URLs
// (CSS/JS). This mimics browser behavior where sub-resources are referenced
// during page load. Responses are discarded. At most maxResourceHints requests
// are made.
func (f *Fetcher) FetchResourceHints(ctx context.Context, pageURL string, resourceURLs []string) {
	if len(resourceURLs) == 0 {
		return
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return
	}

	limit := len(resourceURLs)
	if limit > maxResourceHints {
		limit = maxResourceHints
	}

	for _, rawURL := range resourceURLs[:limit] {
		resolved := resolveURL(base, rawURL)
		if resolved == "" {
			continue
		}

		hctx, cancel := context.WithTimeout(ctx, ResourceHintTimeout)
		req, err := http.NewRequestWithContext(hctx, http.MethodHead, resolved, http.NoBody)
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("User-Agent", f.userAgent)
		req.Header.Set("Referer", pageURL)
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Sec-Fetch-Dest", "style")
		req.Header.Set("Sec-Fetch-Mode", "no-cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")

		resp, err := f.client.Do(req)
		cancel()
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain body before close
		resp.Body.Close()
	}
}

// resolveURL resolves a possibly-relative URL against a base URL.
func resolveURL(base *url.URL, rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	ref, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}
