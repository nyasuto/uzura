// Package network provides HTTP fetching for document loading.
package network

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	uzerr "github.com/nyasuto/uzura/internal/errors"
)

// CookieJar is the interface for cookie storage, matching http.CookieJar.
type CookieJar = http.CookieJar

const (
	// DefaultUserAgent is the default User-Agent header sent with requests.
	DefaultUserAgent = "Uzura/0.1 (+https://github.com/nyasuto/uzura)"

	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 30 * time.Second

	// MaxRedirects is the maximum number of redirects to follow.
	MaxRedirects = 10
)

// FetcherOptions configures a Fetcher.
type FetcherOptions struct {
	UserAgent     string
	Timeout       time.Duration
	EnableCookies bool
	ObeyRobots    bool
	// CookieJar provides an external cookie jar. When set, EnableCookies is
	// ignored and this jar is used directly.
	CookieJar CookieJar
}

// Fetcher performs HTTP requests with redirect tracking and timeouts.
type Fetcher struct {
	client     *http.Client
	userAgent  string
	obeyRobots bool
	robots     robotsCache
}

// NewFetcher creates a new Fetcher with the given options.
// If opts is nil, defaults are used.
func NewFetcher(opts *FetcherOptions) *Fetcher {
	ua := DefaultUserAgent
	timeout := DefaultTimeout

	if opts != nil {
		if opts.UserAgent != "" {
			ua = opts.UserAgent
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
	}

	redirectCount := 0
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > MaxRedirects {
				return fmt.Errorf("%w: stopped after %d", uzerr.ErrTooManyRedirects, MaxRedirects)
			}
			return nil
		},
	}

	if opts != nil && opts.CookieJar != nil {
		client.Jar = opts.CookieJar
	} else if opts != nil && opts.EnableCookies {
		jar, _ := cookiejar.New(nil)
		client.Jar = jar
	}

	obeyRobots := false
	if opts != nil {
		obeyRobots = opts.ObeyRobots
	}

	return &Fetcher{
		client:     client,
		userAgent:  ua,
		obeyRobots: obeyRobots,
		robots:     robotsCache{rules: make(map[string]*robotsRules)},
	}
}

// Fetch retrieves the resource at the given URL.
// The caller must close the response body.
func (f *Fetcher) Fetch(url string) (*http.Response, error) {
	return f.FetchContext(context.Background(), url)
}

// FetchContext retrieves the resource at the given URL with context support.
// The caller must close the response body.
func (f *Fetcher) FetchContext(ctx context.Context, url string) (*http.Response, error) {
	return f.FetchContextWithHeaders(ctx, url, nil)
}

// FetchContextWithHeaders retrieves the resource with optional header overrides.
// Headers in extraHeaders are added to (or override) the default request headers.
// The caller must close the response body.
func (f *Fetcher) FetchContextWithHeaders(ctx context.Context, url string, extraHeaders http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	for k, vals := range extraHeaders {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}

	return resp, nil
}
