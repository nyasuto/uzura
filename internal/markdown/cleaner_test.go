package markdown

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func TestClean_StripScriptStyleNoscript(t *testing.T) {
	const input = `<html><body>
		<p>Keep me</p>
		<script>alert(1)</script>
		<style>.x{}</style>
		<noscript>No JS</noscript>
	</body></html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	Clean(doc, false)

	html := dom.Serialize(doc)
	if strings.Contains(html, "alert") {
		t.Error("script should be removed")
	}
	if strings.Contains(html, ".x{}") {
		t.Error("style should be removed")
	}
	if strings.Contains(html, "No JS") {
		t.Error("noscript should be removed")
	}
	if !strings.Contains(html, "Keep me") {
		t.Error("content should be preserved")
	}
}

func TestClean_ArticleMode_RemovesLayout(t *testing.T) {
	const input = `<html><body>
		<nav>Menu</nav>
		<header>Header</header>
		<article><p>Content</p></article>
		<aside>Sidebar</aside>
		<footer>Footer</footer>
	</body></html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	Clean(doc, true)

	html := dom.Serialize(doc)
	for _, removed := range []string{"Menu", "Header", "Sidebar", "Footer"} {
		if strings.Contains(html, removed) {
			t.Errorf("%s should be removed in article mode", removed)
		}
	}
	if !strings.Contains(html, "Content") {
		t.Error("article content should be preserved")
	}
}

func TestClean_NonArticleMode_KeepsLayout(t *testing.T) {
	const input = `<html><body><nav>Menu</nav><p>Text</p></body></html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	Clean(doc, false)

	html := dom.Serialize(doc)
	if !strings.Contains(html, "Menu") {
		t.Error("nav should be kept in non-article mode")
	}
}

func TestClean_HiddenElements(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"hidden attr", `<html><body><div hidden>Hidden</div><p>Visible</p></body></html>`},
		{"display:none", `<html><body><div style="display:none">Hidden</div><p>Visible</p></body></html>`},
		{"display: none", `<html><body><div style="display: none">Hidden</div><p>Visible</p></body></html>`},
		{"aria-hidden", `<html><body><div aria-hidden="true">Hidden</div><p>Visible</p></body></html>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := htmlparser.Parse(strings.NewReader(tt.input))
			Clean(doc, false)
			html := dom.Serialize(doc)
			if strings.Contains(html, "Hidden") {
				t.Errorf("hidden element should be removed: %s", html)
			}
			if !strings.Contains(html, "Visible") {
				t.Error("visible content should remain")
			}
		})
	}
}

func TestClean_AdElements(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ad class", `<html><body><div class="ad-banner">Ad</div><p>Content</p></body></html>`},
		{"sidebar class", `<html><body><div class="sidebar-widget">Side</div><p>Content</p></body></html>`},
		{"promo id", `<html><body><div id="promo-section">Promo</div><p>Content</p></body></html>`},
		{"sponsor", `<html><body><div class="sponsor-link">Sponsor</div><p>Content</p></body></html>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := htmlparser.Parse(strings.NewReader(tt.input))
			Clean(doc, false)
			html := dom.Serialize(doc)
			if !strings.Contains(html, "Content") {
				t.Error("main content should remain")
			}
		})
	}
}
