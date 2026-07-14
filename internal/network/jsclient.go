package network

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"

	uzerr "github.com/nyasuto/uzura/internal/errors"
)

// NewJSClient returns an *http.Client that shares this Fetcher's cookie jar
// and overall TLS/HTTP2 transport configuration (cloned from the Fetcher's
// own transport), but dials through its own net.Dialer.
//
// If control is non-nil, it is installed as that dialer's Control func,
// which the net package invokes synchronously immediately before the
// low-level connect() call, once network/address have already been
// resolved to a concrete IP by the dialer — for EVERY dial made through
// this client, including each hop of a redirect chain (the client's
// CheckRedirect only counts hops; every hop still dials through this same
// guarded transport). This is the intended enforcement point for
// JS-initiated-request SSRF protection: a pre-request check of the
// original URL string (like checkJSNetworkTarget) can be bypassed by a
// redirect to a private address, or defeated by DNS rebinding (the IP
// re-resolving differently between check time and dial time). Rejecting
// here instead closes both gaps, because the address inspected is exactly
// the one about to be connected to.
//
// If control is nil, dialing behaves like an ordinary client (used for the
// AllowPrivateNetworkJS opt-out path).
//
// Redirects are capped at MaxRedirects, matching the Fetcher's own client.
func (f *Fetcher) NewJSClient(control func(network, address string, c syscall.RawConn) error) *http.Client {
	transport := f.transport.Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   control,
	}).DialContext

	redirectCount := 0
	return &http.Client{
		Transport: transport,
		Timeout:   f.client.Timeout,
		Jar:       f.client.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > MaxRedirects {
				return fmt.Errorf("%w: stopped after %d", uzerr.ErrTooManyRedirects, MaxRedirects)
			}
			return nil
		},
	}
}
