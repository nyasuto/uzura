package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

// ChallengeResult holds cookies obtained by solving a Cloudflare challenge.
type ChallengeResult struct {
	// Cookies contains the cookies extracted from challenge script execution.
	Cookies []*http.Cookie
}

// scriptTagRe matches inline <script> tags and captures their content.
var scriptTagRe = regexp.MustCompile(`(?is)<script[^>]*>([^<]+)</script>`)

// srcAttrRe matches script tags with a src attribute (external scripts).
var srcAttrRe = regexp.MustCompile(`(?i)\bsrc\s*=`)

// extractChallengeScripts extracts inline script bodies from HTML.
// External scripts (with src attribute) are skipped.
func extractChallengeScripts(html string) []string {
	matches := scriptTagRe.FindAllStringSubmatch(html, -1)
	var scripts []string
	for _, m := range matches {
		tag := m[0]
		body := strings.TrimSpace(m[1])
		if body == "" {
			continue
		}
		// Skip external scripts.
		openTag := tag[:strings.Index(tag, ">")+1]
		if srcAttrRe.MatchString(openTag) {
			continue
		}
		scripts = append(scripts, body)
	}
	return scripts
}

// SolveChallenge attempts to solve a Cloudflare JS challenge by executing
// inline scripts in a sandboxed goja VM and capturing document.cookie
// assignments. Returns the extracted cookies.
func SolveChallenge(body string) (*ChallengeResult, error) {
	scripts := extractChallengeScripts(body)
	if len(scripts) == 0 {
		return &ChallengeResult{}, nil
	}

	vm := goja.New()
	var cookies []*http.Cookie

	// Mock document object with cookie setter that captures assignments.
	doc := vm.NewObject()
	_ = doc.DefineAccessorProperty("cookie",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			// Getter: return empty string.
			return vm.ToValue("")
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			// Setter: parse and capture the cookie.
			if len(call.Arguments) > 0 {
				raw := call.Arguments[0].String()
				if c := parseCookieString(raw); c != nil {
					cookies = append(cookies, c)
				}
			}
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, // not enumerable
		goja.FLAG_TRUE,  // configurable
	)
	_ = vm.Set("document", doc)

	// Provide minimal navigator/window stubs to avoid reference errors.
	_ = vm.Set("window", vm.GlobalObject())
	nav := vm.NewObject()
	_ = nav.Set("userAgent", DefaultUserAgent)
	_ = vm.Set("navigator", nav)
	_ = vm.Set("location", vm.NewObject())

	for _, script := range scripts {
		_, _ = vm.RunString(script) // errors in individual scripts are not fatal
	}

	return &ChallengeResult{Cookies: cookies}, nil
}

// parseCookieString parses a "Set-Cookie"-style string like
// "name=value; path=/; domain=.example.com" into an http.Cookie.
func parseCookieString(raw string) *http.Cookie {
	parts := strings.SplitN(raw, ";", 2)
	nv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
	if len(nv) != 2 || nv[0] == "" {
		return nil
	}
	c := &http.Cookie{
		Name:  strings.TrimSpace(nv[0]),
		Value: strings.TrimSpace(nv[1]),
	}
	if len(parts) > 1 {
		attrs := strings.Split(parts[1], ";")
		for _, attr := range attrs {
			kv := strings.SplitN(strings.TrimSpace(attr), "=", 2)
			key := strings.ToLower(strings.TrimSpace(kv[0]))
			val := ""
			if len(kv) == 2 {
				val = strings.TrimSpace(kv[1])
			}
			switch key {
			case "path":
				c.Path = val
			case "domain":
				c.Domain = val
			case "secure":
				c.Secure = true
			case "httponly":
				c.HttpOnly = true
			}
		}
	}
	return c
}

// FetchWithChallengeRetry fetches the URL and, if a Cloudflare challenge is
// detected, attempts to solve it and retry. The fetcher must have a cookie jar
// enabled for cookies to persist across the retry.
func FetchWithChallengeRetry(f *Fetcher, rawURL string) (*http.Response, error) {
	return FetchWithChallengeRetryContext(f, context.Background(), rawURL)
}

// FetchWithChallengeRetryContext is like FetchWithChallengeRetry but with
// context support.
func FetchWithChallengeRetryContext(f *Fetcher, ctx context.Context, rawURL string) (*http.Response, error) {
	resp, err := f.FetchContext(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	cfResult := DetectCloudflareFromHeaders(resp.Header, resp.StatusCode)
	if !cfResult.Detected {
		return resp, nil
	}

	// Read the challenge body to attempt solving.
	bodyBytes, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading challenge body: %w", readErr)
	}

	bodyStr := string(bodyBytes)

	// Verify via HTML content too.
	htmlResult := DetectCloudflareFromHTML(bodyStr)
	if !htmlResult.Detected && !cfResult.Detected {
		// Rebuild the response with the body we already read.
		resp.Body = io.NopCloser(strings.NewReader(bodyStr))
		return resp, nil
	}

	// Attempt to solve the challenge.
	challenge, solveErr := SolveChallenge(bodyStr)
	if solveErr != nil || len(challenge.Cookies) == 0 {
		// Can't solve; return the original challenge response.
		resp.Body = io.NopCloser(strings.NewReader(bodyStr))
		return resp, nil
	}

	// Inject solved cookies into the jar.
	if f.client.Jar != nil {
		u, parseErr := url.Parse(rawURL)
		if parseErr == nil {
			f.client.Jar.SetCookies(u, challenge.Cookies)
		}
	}

	// Retry the request with the new cookies.
	retryResp, retryErr := f.FetchContext(ctx, rawURL)
	if retryErr != nil {
		// Retry failed; return original challenge response.
		resp.Body = io.NopCloser(strings.NewReader(bodyStr))
		return resp, nil
	}

	return retryResp, nil
}
