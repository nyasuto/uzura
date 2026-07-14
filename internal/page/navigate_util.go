package page

import (
	"context"
	"net/http"
	"time"
)

// fetchWithOverrides performs the navigation-time fetch, applying header
// overrides (e.g. Referer, interceptor-provided headers) when present.
func (p *Page) fetchWithOverrides(ctx context.Context, url string, headers http.Header) (*http.Response, error) {
	if headers == nil {
		return p.fetcher.FetchContext(ctx, url)
	}
	return p.fetcher.FetchContextWithHeaders(ctx, url, headers)
}

// mimeFromResponse extracts the MIME type from a response's Content-Type
// header, stripping any parameters (e.g. charset), defaulting to
// "text/html" when the header is absent.
func mimeFromResponse(resp *http.Response) string {
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		return "text/html"
	}
	for i, c := range ct {
		if c == ';' {
			return ct[:i]
		}
		_ = i
	}
	return ct
}

// nowFunc returns a function producing the current time as Unix seconds
// (fractional), matching the timestamp format used by NetworkEvent.
func nowFunc() func() float64 {
	return func() float64 {
		return float64(time.Now().UnixMilli()) / 1000.0
	}
}

// mergeContexts returns a context that is canceled when either parent is done.
func mergeContexts(ctx1, ctx2 context.Context) (context.Context, context.CancelFunc) {
	merged, cancel := context.WithCancel(ctx1)
	go func() {
		select {
		case <-ctx2.Done():
			cancel()
		case <-merged.Done():
		}
	}()
	return merged, cancel
}
