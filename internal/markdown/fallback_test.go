package markdown

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func TestAssessQuality(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    ContentQuality
	}{
		{"empty", "", QualityFailed},
		{"whitespace only", "   \n\t  ", QualityFailed},
		{"very short", "Loading...", QualityPartial},
		{"short placeholder", "Please wait while the page loads.", QualityPartial},
		{"sufficient content", strings.Repeat("This is article content. ", 10), QualityGood},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssessQuality(tt.content)
			if got != tt.want {
				t.Errorf("AssessQuality(%q) = %d, want %d", tt.content, got, tt.want)
			}
		})
	}
}

func TestFindContentRegion(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantTag string
		wantTxt string
	}{
		{
			name:    "prefers main over article",
			html:    `<html><body><main><p>Main content</p></main><article><p>Article content</p></article></body></html>`,
			wantTag: "MAIN",
			wantTxt: "Main content",
		},
		{
			name:    "falls back to article when no main",
			html:    `<html><body><div><p>Sidebar</p></div><article><p>Article content here</p></article></body></html>`,
			wantTag: "ARTICLE",
			wantTxt: "Article content here",
		},
		{
			name:    "falls back to body when no main or article",
			html:    `<html><body><div><p>Page content</p></div></body></html>`,
			wantTag: "BODY",
			wantTxt: "Page content",
		},
		{
			name: "picks largest main when multiple",
			html: `<html><body>
				<main><p>Small</p></main>
				<main><p>This is a much larger main content area with lots of text</p></main>
			</body></html>`,
			wantTag: "MAIN",
			wantTxt: "This is a much larger main content area with lots of text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := mustParse(t, tt.html)
			region := FindContentRegion(doc)
			if region == nil {
				t.Fatal("FindContentRegion returned nil")
			}

			gotTag := region.NodeName()
			if gotTag != tt.wantTag {
				t.Errorf("tag = %q, want %q", gotTag, tt.wantTag)
			}

			gotText := strings.TrimSpace(region.TextContent())
			if !strings.Contains(gotText, tt.wantTxt) {
				t.Errorf("text = %q, want to contain %q", gotText, tt.wantTxt)
			}
		})
	}
}

func TestRenderWithFallback_ReadabilitySuccess(t *testing.T) {
	// A well-structured article page — readability should succeed
	html := `<html><head><title>Test Article</title></head><body>
		<nav><a href="/">Home</a></nav>
		<article>` + strings.Repeat("<p>This is a paragraph of the article content with enough text. </p>\n", 10) + `</article>
		<footer>Copyright 2024</footer>
	</body></html>`

	doc := mustParse(t, html)
	result := RenderWithFallback(doc, "https://example.com/article")

	if !strings.Contains(result, "title: Test Article") {
		t.Error("missing frontmatter title")
	}
	if !strings.Contains(result, "paragraph of the article content") {
		t.Error("missing article content")
	}
}

func TestRenderWithFallback_FallbackToMain(t *testing.T) {
	// A page where readability will likely fail (no clear article structure)
	// but has a <main> element with content
	html := `<html><head><title>App Page</title></head><body>
		<nav><a href="/">Home</a><a href="/about">About</a></nav>
		<main>
			<h1>Dashboard</h1>
			<div><p>Welcome to your dashboard. Here is your data summary with enough content to be meaningful.</p></div>
			<div><p>Section two with more content for the dashboard page layout.</p></div>
		</main>
		<aside><p>Sidebar widget</p></aside>
		<footer><p>Footer info</p></footer>
	</body></html>`

	doc := mustParse(t, html)
	result := RenderWithFallback(doc, "https://example.com/dashboard")

	if !strings.Contains(result, "Dashboard") {
		t.Error("missing main heading")
	}
	if !strings.Contains(result, "Welcome to your dashboard") {
		t.Error("missing main content")
	}
}

func TestRenderWithFallback_FallbackToBody(t *testing.T) {
	// Minimal page with no main/article — should fall back to body
	html := `<html><head><title>Simple</title></head><body>
		<h1>Hello World</h1>
		<p>This is a simple page with no semantic structure at all.</p>
	</body></html>`

	doc := mustParse(t, html)
	result := RenderWithFallback(doc, "https://example.com/simple")

	if !strings.Contains(result, "Hello World") {
		t.Error("missing heading content")
	}
	if !strings.Contains(result, "simple page") {
		t.Error("missing body content")
	}
}

func TestCleanLayoutInRegion(t *testing.T) {
	html := `<html><body><main>
		<nav><a href="/">Nav link</a></nav>
		<h1>Content</h1>
		<p>Real content here</p>
		<footer><p>Footer inside main</p></footer>
	</main></body></html>`

	doc := mustParse(t, html)
	main := doc.GetElementsByTagName("main")
	if len(main) == 0 {
		t.Fatal("no main element")
	}

	cleanLayoutInRegion(main[0])

	text := main[0].TextContent()
	if strings.Contains(text, "Nav link") {
		t.Error("nav should have been removed")
	}
	if strings.Contains(text, "Footer inside main") {
		t.Error("footer should have been removed")
	}
	if !strings.Contains(text, "Real content here") {
		t.Error("content should remain")
	}
}

func mustParse(t *testing.T, htmlStr string) *dom.Document {
	t.Helper()
	doc, err := htmlparser.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return doc
}
