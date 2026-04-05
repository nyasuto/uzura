// Package network provides HTTP fetching for document loading.
package network

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	uzerr "github.com/nyasuto/uzura/internal/errors"
	"golang.org/x/net/http2"
)

// CookieJar is the interface for cookie storage, matching http.CookieJar.
type CookieJar = http.CookieJar

const (
	// DefaultUserAgent is the default User-Agent header sent with requests.
	// Uses a Chrome-like UA string to reduce bot detection.
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 30 * time.Second

	// MaxRedirects is the maximum number of redirects to follow.
	MaxRedirects = 10

	// MaxRetries is the maximum number of retry attempts for transient errors.
	MaxRetries = 2

	// retryBaseDelay is the base delay for exponential backoff between retries.
	retryBaseDelay = 500 * time.Millisecond
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

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	// Enable HTTP/2 support via ALPN negotiation.
	// Error is ignored since it only fails with incompatible transport settings.
	_ = http2.ConfigureTransport(transport)

	redirectCount := 0
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
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
// Transient errors are retried up to MaxRetries times with exponential backoff.
// The caller must close the response body.
func (f *Fetcher) FetchContextWithHeaders(ctx context.Context, url string, extraHeaders http.Header) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * (1 << (attempt - 1)) // 500ms, 1s
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("fetching %s: %w", url, ctx.Err())
			case <-time.After(delay):
			}
		}

		resp, err := f.doFetch(ctx, url, extraHeaders)
		if err == nil {
			// Retry on server errors (503 Service Unavailable, 429 Too Many Requests)
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
				if attempt < MaxRetries {
					io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain body before retry
					resp.Body.Close()
					lastErr = fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
					continue
				}
			}
			return resp, nil
		}

		if !isRetryable(err) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("fetching %s: %w", url, err)
		}
		lastErr = err
	}
	return nil, fmt.Errorf("fetching %s after %d retries: %w", url, MaxRetries, lastErr)
}

func (f *Fetcher) doFetch(ctx context.Context, url string, extraHeaders http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	for k, vals := range extraHeaders {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}

	return f.client.Do(req)
}

// isRetryable returns true for transient network errors worth retrying.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Connection reset, refused, timeout
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "broken pipe")
}
