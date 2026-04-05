//go:build compat

package mcp_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// compatSite defines a site to test for compatibility.
type compatSite struct {
	Name string
	URL  string
}

var compatSites = []compatSite{
	{Name: "HackerNews", URL: "https://news.ycombinator.com/"},
	{Name: "GoDev", URL: "https://go.dev/"},
	{Name: "Wikipedia", URL: "https://en.wikipedia.org/wiki/Web_browser"},
	{Name: "ReactDev", URL: "https://react.dev/"},
	{Name: "StackOverflow", URL: "https://stackoverflow.com/"},
}

func TestCompat_BrowseText(t *testing.T) {
	p := startMCPWithTimeout(t, 60*time.Second)
	p.initialize(t)

	for _, site := range compatSites {
		t.Run(site.Name, func(t *testing.T) {
			result := p.callTool(t, "browse", map[string]any{
				"url":    site.URL,
				"format": "text",
			})
			text := result.Text()
			if text == "" {
				t.Errorf("browse text returned empty for %s", site.URL)
			}
			if result.IsError {
				t.Logf("WARNING: browse text returned error for %s: %s", site.Name, truncate(text, 200))
			}
		})
	}
}

func TestCompat_BrowseMarkdown(t *testing.T) {
	p := startMCPWithTimeout(t, 60*time.Second)
	p.initialize(t)

	for _, site := range compatSites {
		t.Run(site.Name, func(t *testing.T) {
			result := p.callTool(t, "browse", map[string]any{
				"url":    site.URL,
				"format": "markdown",
			})
			text := result.Text()
			if text == "" {
				t.Errorf("browse markdown returned empty for %s", site.URL)
			}
			if result.IsError {
				t.Logf("WARNING: browse markdown returned error for %s: %s", site.Name, truncate(text, 200))
			}
		})
	}
}

func TestCompat_SemanticTree(t *testing.T) {
	p := startMCPWithTimeout(t, 60*time.Second)
	p.initialize(t)

	for _, site := range compatSites {
		t.Run(site.Name, func(t *testing.T) {
			result := p.callTool(t, "semantic_tree", map[string]any{
				"url": site.URL,
			})
			text := result.Text()
			if text == "" {
				t.Errorf("semantic_tree returned empty for %s", site.URL)
			}
			if result.IsError {
				t.Logf("WARNING: semantic_tree returned error for %s: %s", site.Name, truncate(text, 200))
			}
		})
	}
}

func TestCompat_QueryH1(t *testing.T) {
	p := startMCPWithTimeout(t, 60*time.Second)
	p.initialize(t)

	for _, site := range compatSites {
		t.Run(site.Name, func(t *testing.T) {
			result := p.callTool(t, "query", map[string]any{
				"url":      site.URL,
				"selector": "h1",
			})
			text := result.Text()
			if result.IsError {
				t.Logf("WARNING: query h1 returned error for %s: %s", site.Name, truncate(text, 200))
				return
			}
			// Parse response to check total.
			var resp struct {
				Total int `json:"total"`
			}
			if err := json.Unmarshal([]byte(text), &resp); err != nil {
				t.Logf("WARNING: could not parse query response for %s: %v", site.Name, err)
				return
			}
			// Some sites may not have h1 (e.g., Cloudflare blocked sites).
			// Just log it rather than failing.
			if resp.Total == 0 {
				// Check if it might be Cloudflare blocked.
				browseResult := p.callTool(t, "browse", map[string]any{
					"url":    site.URL,
					"format": "text",
				})
				browseText := browseResult.Text()
				if strings.Contains(strings.ToLower(browseText), "cloudflare") {
					t.Logf("INFO: %s has no h1 (likely Cloudflare blocked)", site.Name)
				} else {
					t.Logf("INFO: %s has no h1 element", site.Name)
				}
			}
		})
	}
}
