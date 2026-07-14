package page

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
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
