package page

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"

	"github.com/nyasuto/uzura/internal/network"
)

// isPrivateOrLocalAddr reports whether addr is a destination JS-initiated
// requests (fetch/XMLHttpRequest) must not reach by default: loopback,
// link-local (including the 169.254.169.254 cloud metadata endpoint),
// RFC1918 private ranges, IPv6 unique-local addresses, the unspecified
// address, or multicast.
func isPrivateOrLocalAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsPrivate() ||
		addr.IsUnspecified() ||
		addr.IsMulticast()
}

// checkJSNetworkTarget blocks JS-initiated requests (fetch/XMLHttpRequest)
// to private/internal destinations, mitigating SSRF: an untrusted browsed
// page must not be able to fetch cloud metadata endpoints, loopback, or
// RFC1918-internal services and exfiltrate the response in-VM.
//
// If rawURL's host is an IP literal, it is classified directly. If it's a
// hostname, it is resolved via net.DefaultResolver.LookupIPAddr and blocked
// if ANY resolved address is private — this closes the "localhost" /
// DNS-rebinding gap a hostname-string-only check would miss. Resolution
// failure fails closed (blocks), since an address that can't be verified
// safe must not be treated as safe.
func checkJSNetworkTarget(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("js network request blocked: invalid URL %q: %w", rawURL, err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("js network request blocked: missing host in %q", rawURL)
	}

	if addr, perr := netip.ParseAddr(host); perr == nil {
		if isPrivateOrLocalAddr(addr) {
			return fmt.Errorf("js network request blocked: %s is a private/internal address", host)
		}
		return nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("js network request blocked: could not resolve host %q: %w", host, err)
	}
	for _, a := range addrs {
		na, ok := netip.AddrFromSlice(a.IP)
		if !ok {
			continue
		}
		if isPrivateOrLocalAddr(na) {
			return fmt.Errorf("js network request blocked: host %q resolves to private/internal address %s", host, na)
		}
	}
	return nil
}

// ssrfDialControl is the REAL enforcement point for JS-initiated-request
// SSRF protection. It matches the net.Dialer.Control signature and is
// installed via Fetcher.NewJSClient, so it runs synchronously immediately
// before the low-level connect() call, for every dial made through that
// client — the initial connection and every redirect hop alike — against
// address, which the dialer has already resolved to a concrete IP:port.
//
// checkJSNetworkTarget (the pre-request check on the original URL) only
// runs once and never sees where a redirect ultimately leads or what a
// hostname later re-resolves to; a 302 to a private address, or a
// DNS-rebound name, sails through it untouched. Rejecting here instead
// closes both gaps, because the address inspected is exactly the one about
// to be connected to, at the moment it's about to be connected to.
//
// The syscall.RawConn parameter (the raw OS connection, for callers that
// need to twiddle socket options) is unused here — the guard only needs
// the address.
func ssrfDialControl(dialNetwork, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("js network request blocked: invalid dial address %q for %s: %w", address, dialNetwork, err)
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("js network request blocked: could not parse dial IP %q: %w", host, err)
	}
	if isPrivateOrLocalAddr(addr) {
		return fmt.Errorf("js network request blocked: dial target %s is a private/internal address", addr)
	}
	return nil
}

// newJSClientFor builds the *http.Client a Page uses for all JS-initiated
// requests (fetch/XMLHttpRequest). It shares f's cookie jar and TLS/HTTP2
// transport configuration (via network.Fetcher.NewJSClient), and — unless
// allowPrivateJS is true — installs ssrfDialControl so every dial made
// through the client, including each hop of a redirect chain, is rejected
// if it targets a private/internal address. That per-dial enforcement is
// what closes the SSRF bypass a one-time pre-request URL check
// (checkJSNetworkTarget) alone would leave open.
func newJSClientFor(f *network.Fetcher, allowPrivateJS bool) *http.Client {
	if allowPrivateJS {
		return f.NewJSClient(nil)
	}
	return f.NewJSClient(ssrfDialControl)
}
