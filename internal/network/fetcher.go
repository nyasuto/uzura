// Package network provides HTTP fetching for document loading.
package network

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"
)

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
}

// Fetcher performs HTTP requests with redirect tracking and timeouts.
type Fetcher struct {
	client    *http.Client
	userAgent string
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
				return fmt.Errorf("stopped after %d redirects", MaxRedirects)
			}
			return nil
		},
	}

	if opts != nil && opts.EnableCookies {
		jar, _ := cookiejar.New(nil)
		client.Jar = jar
	}

	return &Fetcher{
		client:    client,
		userAgent: ua,
	}
}

// Fetch retrieves the resource at the given URL.
// The caller must close the response body.
func (f *Fetcher) Fetch(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}

	return resp, nil
}
