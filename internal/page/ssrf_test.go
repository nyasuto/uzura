package page

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/nyasuto/uzura/internal/network"
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

// TestDialGuardRejectsPrivateIP is a table-driven test for ssrfDialControl,
// the net.Dialer.Control-shaped function that is the REAL enforcement point
// for JS-initiated-request SSRF protection: unlike checkJSNetworkTarget
// (which only inspects the original URL once, before any redirects), this
// runs at dial time, on every connection attempt (initial + each redirect
// hop), against the address the dialer actually resolved and is about to
// connect to.
func TestDialGuardRejectsPrivateIP(t *testing.T) {
	tests := []struct {
		network string
		address string
		wantErr bool
	}{
		{"tcp4", "127.0.0.1:80", true},
		{"tcp4", "169.254.169.254:80", true},
		{"tcp4", "10.0.0.1:80", true},
		{"tcp6", "[::1]:80", true},
		{"tcp6", "[fc00::1]:80", true},
		{"tcp4", "8.8.8.8:80", false},
		{"tcp6", "[2606:2800::1]:80", false},
	}
	for _, tt := range tests {
		t.Run(tt.network+"/"+tt.address, func(t *testing.T) {
			err := ssrfDialControl(tt.network, tt.address, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ssrfDialControl(%q, %q) error = %v, wantErr %v", tt.network, tt.address, err, tt.wantErr)
			}
		})
	}
}

// TestJSClient_DialGuardBlocksLoopback proves the dial-time guard rejects a
// connection to a loopback destination at the ACTUAL dial, not just via a
// pre-request URL string check. This is what closes both the redirect
// bypass (every redirect hop dials through this same guarded transport) and
// the DNS-rebinding TOCTOU (the IP checked here is the exact IP about to be
// connected to).
//
// Before the dial-time guard existed, only a one-time pre-request check on
// the original URL existed (checkJSNetworkTarget in jsHTTPClient), which a
// 302 redirect to a private address would sail through untouched. This test
// exercises the guarded *http.Client directly, independent of any redirect
// plumbing, so it fails to compile/RED until Fetcher.NewJSClient exists.
func TestJSClient_DialGuardBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "should never be reached")
	}))
	defer srv.Close()

	f := network.NewFetcher(nil)
	client := f.NewJSClient(ssrfDialControl)

	_, err := client.Get(srv.URL)
	if err == nil {
		t.Fatal("want error dialing loopback httptest server through guarded client, got nil")
	}
	if !strings.Contains(err.Error(), "private/internal") {
		t.Errorf("err = %v, want it to mention private/internal address", err)
	}
}

// TestJSClient_NoGuardAllowsLoopback confirms that when no dial control is
// installed (the AllowPrivateNetworkJS opt-out path), the client built by
// NewJSClient behaves like an ordinary HTTP client and can reach loopback
// destinations, including following redirects across two loopback servers
// (proving the guard's absence doesn't otherwise regress redirect-following
// behavior).
func TestJSClient_NoGuardAllowsLoopback(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "final destination")
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	f := network.NewFetcher(nil)
	client := f.NewJSClient(nil)

	resp, err := client.Get(redirector.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (redirect should have been followed to final)", resp.StatusCode)
	}
}

// TestNavigate_JSFetchFollowsRedirectWhenAllowed is the full Page/Navigate/JS
// integration-level counterpart of TestJSClient_NoGuardAllowsLoopback: with
// AllowPrivateNetworkJS true (guard disabled), a script's fetch() to a
// server that 302-redirects to a second server must still transparently
// follow the redirect and see the final response, proving the SSRF-guard
// work in this file does not regress ordinary multi-hop JS fetches when the
// opt-out is set.
func TestNavigate_JSFetchFollowsRedirectWhenAllowed(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"redirected title"}`)
	}))
	defer final.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/final", http.StatusFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body><div id="root">Loading...</div>
<script>
fetch("/api/post").then(function(r){ return r.json(); }).then(function(data){
  var h1 = document.createElement("h1");
  h1.textContent = data.title;
  document.getElementById("root").textContent = "";
  document.getElementById("root").appendChild(h1);
}).catch(function(e){
  document.getElementById("root").textContent = "blocked: " + e;
});
</script></body></html>`)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	p := New(&Options{AllowPrivateNetworkJS: true})
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.Navigate(ctx, ts.URL); err != nil {
		t.Fatal(err)
	}

	h1, err := p.Document().QuerySelector("h1")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == nil {
		t.Fatal("h1 not found: redirect chain was not followed")
	}
	if h1.TextContent() != "redirected title" {
		t.Errorf("h1 = %q, want %q", h1.TextContent(), "redirected title")
	}
}
