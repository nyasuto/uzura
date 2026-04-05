package network

import (
	"net/http"
	"testing"
)

func TestDetectCloudflareFromHeaders(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		statusCode int
		wantDetect bool
		wantReason string
	}{
		{
			name:       "nil headers",
			headers:    nil,
			statusCode: 200,
			wantDetect: false,
		},
		{
			name:       "normal 200 response",
			headers:    http.Header{"Content-Type": {"text/html"}},
			statusCode: 200,
			wantDetect: false,
		},
		{
			name:       "cf-ray with 200 is not a challenge",
			headers:    http.Header{"Cf-Ray": {"abc123"}},
			statusCode: 200,
			wantDetect: false,
		},
		{
			name:       "cf-mitigated header",
			headers:    http.Header{"Cf-Mitigated": {"challenge"}},
			statusCode: 403,
			wantDetect: true,
			wantReason: "cf-mitigated header: challenge",
		},
		{
			name:       "403 with cf-ray",
			headers:    http.Header{"Cf-Ray": {"abc123-NRT"}},
			statusCode: 403,
			wantDetect: true,
			wantReason: "403 with cf-ray header",
		},
		{
			name: "503 with cf-ray and server cloudflare",
			headers: http.Header{
				"Cf-Ray": {"def456-LAX"},
				"Server": {"cloudflare"},
			},
			statusCode: 503,
			wantDetect: true,
			wantReason: "503 with cf-ray and server: cloudflare",
		},
		{
			name:       "503 without cf-ray is not cloudflare",
			headers:    http.Header{"Server": {"nginx"}},
			statusCode: 503,
			wantDetect: false,
		},
		{
			name:       "403 without cf-ray is not cloudflare",
			headers:    http.Header{"Server": {"Apache"}},
			statusCode: 403,
			wantDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectCloudflareFromHeaders(tt.headers, tt.statusCode)
			if result.Detected != tt.wantDetect {
				t.Errorf("Detected = %v, want %v", result.Detected, tt.wantDetect)
			}
			if tt.wantReason != "" && result.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestDetectCloudflareFromHTML(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantDetect bool
	}{
		{
			name:       "empty body",
			body:       "",
			wantDetect: false,
		},
		{
			name:       "normal HTML page",
			body:       "<html><body><h1>Hello World</h1></body></html>",
			wantDetect: false,
		},
		{
			name:       "enable javascript and cookies",
			body:       `<html><body><p>Please Enable JavaScript and cookies to continue</p></body></html>`,
			wantDetect: true,
		},
		{
			name:       "checking your browser",
			body:       `<title>Checking your browser before accessing example.com</title>`,
			wantDetect: true,
		},
		{
			name:       "just a moment title",
			body:       `<title>Just a moment...</title><body>Please wait</body>`,
			wantDetect: true,
		},
		{
			name:       "attention required cloudflare",
			body:       `<title>Attention Required! | Cloudflare</title>`,
			wantDetect: true,
		},
		{
			name:       "verify you are human",
			body:       `<div>Verify you are human by completing the action below.</div>`,
			wantDetect: true,
		},
		{
			name:       "challenges.cloudflare.com iframe",
			body:       `<iframe src="https://challenges.cloudflare.com/cdn-cgi/challenge-platform/..."></iframe>`,
			wantDetect: true,
		},
		{
			name:       "cf-browser-verification div",
			body:       `<div id="cf-browser-verification"><p>Processing...</p></div>`,
			wantDetect: true,
		},
		{
			name:       "cf_chl_opt script variable",
			body:       `<script>var cf_chl_opt = {cvId: "2", cZone: "example.com"};</script>`,
			wantDetect: true,
		},
		{
			name:       "ray id in footer",
			body:       `<div class="cf-footer"><span>Ray ID: abc123def456</span></div>`,
			wantDetect: true,
		},
		{
			name:       "case insensitive detection",
			body:       `<title>JUST A MOMENT...</title>`,
			wantDetect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectCloudflareFromHTML(tt.body)
			if result.Detected != tt.wantDetect {
				t.Errorf("Detected = %v, want %v (body: %s)", result.Detected, tt.wantDetect, tt.body)
			}
		})
	}
}

func TestDetectCloudflare(t *testing.T) {
	t.Run("header detection takes priority", func(t *testing.T) {
		headers := http.Header{"Cf-Mitigated": {"challenge"}}
		body := `<html><body>Normal content</body></html>`
		result := DetectCloudflare(headers, 403, body)
		if !result.Detected {
			t.Error("expected Detected = true")
		}
		if result.Reason != "cf-mitigated header: challenge" {
			t.Errorf("expected header-based reason, got %q", result.Reason)
		}
	})

	t.Run("falls back to HTML detection", func(t *testing.T) {
		headers := http.Header{"Content-Type": {"text/html"}}
		body := `<title>Just a moment...</title>`
		result := DetectCloudflare(headers, 200, body)
		if !result.Detected {
			t.Error("expected Detected = true")
		}
		if result.Reason != "html contains: just a moment..." {
			t.Errorf("expected HTML-based reason, got %q", result.Reason)
		}
	})

	t.Run("no detection for clean page", func(t *testing.T) {
		headers := http.Header{"Content-Type": {"text/html"}}
		body := `<html><body><h1>Welcome</h1></body></html>`
		result := DetectCloudflare(headers, 200, body)
		if result.Detected {
			t.Error("expected Detected = false for clean page")
		}
	})
}
