package markdown

import (
	"strings"
	"testing"

	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func TestExtractMetadata_Basic(t *testing.T) {
	const input = `<!DOCTYPE html>
<html>
<head>
<title>My Page</title>
<meta name="description" content="A page about things">
<meta name="author" content="Jane Doe">
</head>
<body><p>Content</p></body>
</html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	m := ExtractMetadata(doc, "https://example.com/page")

	if m.Title != "My Page" {
		t.Errorf("title = %q, want %q", m.Title, "My Page")
	}
	if m.Description != "A page about things" {
		t.Errorf("description = %q", m.Description)
	}
	if m.Author != "Jane Doe" {
		t.Errorf("author = %q", m.Author)
	}
	if m.URL != "https://example.com/page" {
		t.Errorf("url = %q", m.URL)
	}
}

func TestExtractMetadata_OpenGraph(t *testing.T) {
	const input = `<!DOCTYPE html>
<html>
<head>
<title>Fallback Title</title>
<meta property="og:title" content="OG Title">
<meta property="og:description" content="OG Description">
<meta property="og:image" content="https://example.com/img.jpg">
</head>
<body></body>
</html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	m := ExtractMetadata(doc, "https://example.com")

	if m.OGTitle != "OG Title" {
		t.Errorf("og:title = %q", m.OGTitle)
	}
	if m.OGDesc != "OG Description" {
		t.Errorf("og:description = %q", m.OGDesc)
	}
	if m.OGImage != "https://example.com/img.jpg" {
		t.Errorf("og:image = %q", m.OGImage)
	}
}

func TestExtractMetadata_JSONLD(t *testing.T) {
	const input = `<!DOCTYPE html>
<html>
<head>
<title></title>
<script type="application/ld+json">
{
  "@type": "Article",
  "headline": "JSON-LD Title",
  "description": "JSON-LD Description",
  "author": {"@type": "Person", "name": "John Smith"}
}
</script>
</head>
<body></body>
</html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	m := ExtractMetadata(doc, "https://example.com")

	if m.Title != "JSON-LD Title" {
		t.Errorf("title = %q, want %q", m.Title, "JSON-LD Title")
	}
	if m.Description != "JSON-LD Description" {
		t.Errorf("description = %q", m.Description)
	}
	if m.Author != "John Smith" {
		t.Errorf("author = %q", m.Author)
	}
}

func TestExtractMetadata_NoMetadata(t *testing.T) {
	const input = `<!DOCTYPE html><html><body><p>Hello</p></body></html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	m := ExtractMetadata(doc, "https://example.com")

	if m.URL != "https://example.com" {
		t.Errorf("url should always be set: %q", m.URL)
	}
}

func TestExtractMetadata_PartialMetadata(t *testing.T) {
	const input = `<!DOCTYPE html>
<html>
<head><title>Only Title</title></head>
<body></body>
</html>`

	doc, _ := htmlparser.Parse(strings.NewReader(input))
	m := ExtractMetadata(doc, "https://example.com")

	if m.Title != "Only Title" {
		t.Errorf("title = %q", m.Title)
	}
	if m.Author != "" {
		t.Errorf("author should be empty: %q", m.Author)
	}
}

func TestFormatFrontmatter(t *testing.T) {
	m := &Metadata{
		Title:  "Test Title",
		Author: "Test Author",
		URL:    "https://example.com",
	}

	got := FormatFrontmatter(m)
	if !strings.HasPrefix(got, "---\n") {
		t.Error("should start with ---")
	}
	if !strings.HasSuffix(got, "---\n") {
		t.Error("should end with ---")
	}
	if !strings.Contains(got, "title: Test Title") {
		t.Errorf("missing title: %q", got)
	}
	if !strings.Contains(got, "author: Test Author") {
		t.Errorf("missing author: %q", got)
	}
}

func TestFormatFrontmatter_OGPriority(t *testing.T) {
	m := &Metadata{
		Title:   "HTML Title",
		OGTitle: "OG Title",
		URL:     "https://example.com",
	}

	got := FormatFrontmatter(m)
	if !strings.Contains(got, "title: OG Title") {
		t.Errorf("OG title should take priority: %q", got)
	}
}
