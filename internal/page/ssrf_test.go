package page

import (
	"context"
	"net/netip"
	"strings"
	"testing"
)

// TestIsPrivateOrLocalAddr is a table-driven test for the address
// classification helper that backs JS-initiated request filtering
// (fetch/XHR SSRF protection). It must block loopback, link-local
// (including the 169.254.169.254 cloud metadata endpoint), RFC1918
// private ranges, and IPv6 unique-local addresses, while allowing
// ordinary public addresses.
func TestIsPrivateOrLocalAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.53", true},
		{"169.254.169.254", true}, // cloud metadata endpoint
		{"169.254.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"::1", true},
		{"fc00::1", true},   // unique-local IPv6
		{"fe80::1", true},   // link-local IPv6
		{"0.0.0.0", true},   // unspecified
		{"::", true},        // unspecified
		{"224.0.0.1", true}, // multicast
		{"8.8.8.8", false},
		{"93.184.216.34", false},
		{"1.1.1.1", false},
		{"172.32.0.1", false}, // just outside 172.16/12
		{"2001:4860:4860::8888", false},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			addr, err := netip.ParseAddr(tt.addr)
			if err != nil {
				t.Fatalf("ParseAddr(%q): %v", tt.addr, err)
			}
			if got := isPrivateOrLocalAddr(addr); got != tt.want {
				t.Errorf("isPrivateOrLocalAddr(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

// TestCheckJSNetworkTarget_IPLiteral exercises the direct-IP-literal path
// (no DNS resolution involved), which classification alone can't fully
// cover since it also validates URL parsing and error wrapping.
func TestCheckJSNetworkTarget_IPLiteral(t *testing.T) {
	ctx := context.Background()

	if err := checkJSNetworkTarget(ctx, "http://169.254.169.254/latest/meta-data/"); err == nil {
		t.Error("expected block for cloud metadata IP literal, got nil error")
	}
	if err := checkJSNetworkTarget(ctx, "http://127.0.0.1:8080/"); err == nil {
		t.Error("expected block for loopback IP literal, got nil error")
	}
	if err := checkJSNetworkTarget(ctx, "http://[::1]/"); err == nil {
		t.Error("expected block for IPv6 loopback literal, got nil error")
	}
	if err := checkJSNetworkTarget(ctx, "http://8.8.8.8/"); err != nil {
		t.Errorf("expected public IP literal to be allowed, got error: %v", err)
	}
}

// TestCheckJSNetworkTarget_Hostname confirms hostname resolution blocks
// "localhost" (resolves to a loopback address) — the localhost/rebinding
// gap a naive string-match-only filter would miss.
func TestCheckJSNetworkTarget_Hostname(t *testing.T) {
	err := checkJSNetworkTarget(context.Background(), "http://localhost:1234/")
	if err == nil {
		t.Fatal("expected localhost to be blocked (resolves to loopback), got nil error")
	}
	if !strings.Contains(err.Error(), "localhost") && !strings.Contains(err.Error(), "127.0.0.1") && !strings.Contains(err.Error(), "::1") {
		t.Errorf("error should mention the blocked host/address, got: %v", err)
	}
}

// TestCheckJSNetworkTarget_ResolutionFailure confirms fail-closed behavior:
// an unresolvable hostname is blocked, not allowed through.
func TestCheckJSNetworkTarget_ResolutionFailure(t *testing.T) {
	err := checkJSNetworkTarget(context.Background(), "http://this-host-should-not-resolve.invalid/")
	if err == nil {
		t.Fatal("expected block on DNS resolution failure (fail closed), got nil error")
	}
}
