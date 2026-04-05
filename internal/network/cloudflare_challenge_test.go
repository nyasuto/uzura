package network

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractChallengeScripts(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		wantCount int
	}{
		{
			name:      "no scripts",
			html:      `<html><body>Hello</body></html>`,
			wantCount: 0,
		},
		{
			name:      "inline script",
			html:      `<html><script>document.cookie="__cf_bm=abc";</script></html>`,
			wantCount: 1,
		},
		{
			name:      "external script is skipped",
			html:      `<html><script src="https://example.com/a.js"></script></html>`,
			wantCount: 0,
		},
		{
			name: "multiple scripts",
			html: `<html>
				<script>var a = 1;</script>
				<script>var b = 2;</script>
			</html>`,
			wantCount: 2,
		},
		{
			name:      "empty script",
			html:      `<html><script></script></html>`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scripts := extractChallengeScripts(tt.html)
			if len(scripts) != tt.wantCount {
				t.Errorf("got %d scripts, want %d", len(scripts), tt.wantCount)
			}
		})
	}
}

func TestSolveChallenge(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantCookies int
		wantErr     bool
	}{
		{
			name:        "simple cookie assignment",
			html:        `<html><script>document.cookie="__cf_bm=test123; path=/";</script></html>`,
			wantCookies: 1,
		},
		{
			name: "computed cookie value",
			html: `<html><script>
				var a = 10 + 20;
				document.cookie = "__cf_bm=" + a + "; path=/";
			</script></html>`,
			wantCookies: 1,
		},
		{
			name:        "no cookie assignment",
			html:        `<html><script>var x = 42;</script></html>`,
			wantCookies: 0,
		},
		{
			name: "multiple cookie assignments",
			html: `<html><script>
				document.cookie = "__cf_bm=abc; path=/";
				document.cookie = "cf_clearance=xyz; path=/";
			</script></html>`,
			wantCookies: 2,
		},
		{
			name:        "no scripts at all",
			html:        `<html><body>Not a challenge</body></html>`,
			wantCookies: 0,
		},
		{
			name:        "script with syntax error does not fail entirely",
			html:        `<html><script>document.cookie="a=1; path=/";</script><script>this is bad syntax {{{{</script></html>`,
			wantCookies: 1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SolveChallenge(tt.html)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Cookies) != tt.wantCookies {
				t.Errorf("got %d cookies, want %d; cookies=%v", len(result.Cookies), tt.wantCookies, result.Cookies)
			}
		})
	}
}

func TestSolveChallengeCookieValues(t *testing.T) {
	html := `<html><script>
		var val = String.fromCharCode(104, 101, 108, 108, 111);
		document.cookie = "__cf_bm=" + val + "; path=/; domain=.example.com";
	</script></html>`

	result, err := SolveChallenge(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(result.Cookies))
	}
	c := result.Cookies[0]
	if c.Name != "__cf_bm" {
		t.Errorf("cookie name = %q, want __cf_bm", c.Name)
	}
	if c.Value != "hello" {
		t.Errorf("cookie value = %q, want hello", c.Value)
	}
}

func TestFetchWithChallengeRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the cookie is present, return success.
		cookie, err := r.Cookie("__cf_bm")
		if err == nil && cookie.Value == "solved_token" {
			w.WriteHeader(200)
			fmt.Fprint(w, `<html><body>Success!</body></html>`)
			return
		}
		// Otherwise, always return a Cloudflare challenge.
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("Cf-Ray", "abc123")
		w.WriteHeader(403)
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Attention Required! | Cloudflare</title></head>
<body>
<div id="cf-browser-verification">
<script>document.cookie="__cf_bm=solved_token; path=/";</script>
</div>
</body></html>`)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	fetcher := NewFetcher(&FetcherOptions{
		EnableCookies: true,
		CookieJar:     jar,
	})

	resp, err := FetchWithChallengeRetry(fetcher, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d; body=%s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Success!") {
		t.Errorf("expected body to contain 'Success!', got %q", string(body))
	}
}

func TestFetchWithChallengeRetryNonChallenge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `<html><body>Normal page</body></html>`)
	}))
	defer srv.Close()

	fetcher := NewFetcher(&FetcherOptions{EnableCookies: true})
	resp, err := FetchWithChallengeRetry(fetcher, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestFetchWithChallengeRetryUnsolvable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return a challenge with no solvable script
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("Cf-Ray", "abc123")
		w.WriteHeader(403)
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Attention Required! | Cloudflare</title></head>
<body><div id="cf-browser-verification">
<p>Please enable JavaScript and cookies to continue</p>
</div></body></html>`)
	}))
	defer srv.Close()

	fetcher := NewFetcher(&FetcherOptions{EnableCookies: true})
	resp, err := FetchWithChallengeRetry(fetcher, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// Should return the original challenge response when unsolvable
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 for unsolvable challenge, got %d", resp.StatusCode)
	}
}
