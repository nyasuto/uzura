package network

import (
	"net/http"
	"strings"
)

// CloudflareResult holds the result of Cloudflare challenge detection.
type CloudflareResult struct {
	// Detected is true when the response appears to be a Cloudflare challenge page.
	Detected bool
	// Reason describes why the page was identified as a Cloudflare challenge.
	Reason string
}

// challengeHTMLIndicators are text patterns found in Cloudflare challenge pages.
var challengeHTMLIndicators = []string{
	"enable javascript and cookies to continue",
	"checking your browser before accessing",
	"just a moment...",
	"attention required! | cloudflare",
	"please wait while we verify your browser",
	"verify you are human",
	"challenges.cloudflare.com",
	"cf-browser-verification",
	"cf_chl_opt",
	"ray id:",
}

// DetectCloudflareFromHeaders checks HTTP response headers for signs of
// a Cloudflare challenge or block page.
func DetectCloudflareFromHeaders(headers http.Header, statusCode int) *CloudflareResult {
	if headers == nil {
		return &CloudflareResult{}
	}

	// cf-mitigated header explicitly indicates Cloudflare mitigation.
	if v := headers.Get("Cf-Mitigated"); v != "" {
		return &CloudflareResult{
			Detected: true,
			Reason:   "cf-mitigated header: " + v,
		}
	}

	// A 403 with cf-ray header is a strong signal of a Cloudflare block.
	hasCfRay := headers.Get("Cf-Ray") != ""
	if hasCfRay && statusCode == http.StatusForbidden {
		return &CloudflareResult{
			Detected: true,
			Reason:   "403 with cf-ray header",
		}
	}

	// A 503 with cf-ray and server: cloudflare indicates a challenge page.
	server := strings.ToLower(headers.Get("Server"))
	if hasCfRay && statusCode == http.StatusServiceUnavailable && strings.Contains(server, "cloudflare") {
		return &CloudflareResult{
			Detected: true,
			Reason:   "503 with cf-ray and server: cloudflare",
		}
	}

	return &CloudflareResult{}
}

// DetectCloudflareFromHTML checks the HTML body content for Cloudflare
// challenge page indicators.
func DetectCloudflareFromHTML(body string) *CloudflareResult {
	lower := strings.ToLower(body)

	for _, indicator := range challengeHTMLIndicators {
		if strings.Contains(lower, indicator) {
			return &CloudflareResult{
				Detected: true,
				Reason:   "html contains: " + indicator,
			}
		}
	}

	return &CloudflareResult{}
}

// DetectCloudflare performs both header-based and content-based detection,
// returning the first positive result.
func DetectCloudflare(headers http.Header, statusCode int, body string) *CloudflareResult {
	if r := DetectCloudflareFromHeaders(headers, statusCode); r.Detected {
		return r
	}
	return DetectCloudflareFromHTML(body)
}
