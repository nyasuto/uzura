package markdown

import (
	"strings"
	"testing"

	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func TestExtract_ArticlePage(t *testing.T) {
	const articleHTML = `<!DOCTYPE html>
<html>
<head><title>Test Article Title</title></head>
<body>
<header><nav>Menu</nav></header>
<article>
<h1>Test Article Title</h1>
<p>This is the first paragraph of a test article. It contains enough text
to be recognized as readable content by the readability algorithm.
We need a reasonable amount of content here for extraction to succeed.</p>
<p>This is the second paragraph with more content. The readability library
needs sufficient text density to identify the main content area. Adding
more sentences helps ensure reliable extraction across different heuristics.</p>
<p>Third paragraph continues the article with additional information that
makes this look like a real article rather than a navigation page or a
simple list. Real articles typically have multiple paragraphs of text.</p>
</article>
<footer>Copyright 2026</footer>
</body>
</html>`

	doc, err := htmlparser.Parse(strings.NewReader(articleHTML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Extract(doc, "https://example.com/article")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if result.Title == "" {
		t.Error("expected non-empty title")
	}

	if result.Content == "" {
		t.Error("expected non-empty content")
	}

	// Content should contain article text
	if !strings.Contains(result.Content, "first paragraph") {
		t.Error("content should contain article text")
	}

	// Content should not contain nav/footer
	if strings.Contains(result.Content, "Copyright 2026") {
		t.Error("content should not contain footer text")
	}
}

func TestExtract_NonArticlePage(t *testing.T) {
	const navHTML = `<!DOCTYPE html>
<html>
<head><title>Homepage</title></head>
<body>
<nav>
<a href="/a">Link A</a>
<a href="/b">Link B</a>
<a href="/c">Link C</a>
</nav>
</body>
</html>`

	doc, err := htmlparser.Parse(strings.NewReader(navHTML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Non-article pages may return an error or empty content
	result, err := Extract(doc, "https://example.com/")
	if err != nil {
		// Expected: readability may fail on non-article pages
		return
	}

	// If no error, content should be minimal
	if len(result.Content) > 500 {
		t.Errorf("non-article page should have minimal content, got %d bytes", len(result.Content))
	}
}

func TestExtract_Metadata(t *testing.T) {
	const metaHTML = `<!DOCTYPE html>
<html>
<head>
<title>Meta Test</title>
<meta name="author" content="John Doe">
<meta name="description" content="A test article about metadata">
</head>
<body>
<article>
<h1>Meta Test</h1>
<p>By John Doe</p>
<p>This is a test article with metadata. It needs enough content for
readability to recognize it as a valid article. More text helps ensure
the extraction algorithm identifies this as the main content area.</p>
<p>Additional paragraph to increase the text density and make the
readability heuristics more confident in their identification of the
main content block within this document structure.</p>
<p>Yet another paragraph for good measure. The more substantive content
we have, the more reliable the readability extraction becomes.</p>
</article>
</body>
</html>`

	doc, err := htmlparser.Parse(strings.NewReader(metaHTML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Extract(doc, "https://example.com/meta")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if result.Title != "Meta Test" {
		t.Errorf("title = %q, want %q", result.Title, "Meta Test")
	}
}

func TestIsReadable_Article(t *testing.T) {
	const articleHTML = `<!DOCTYPE html>
<html><head><title>Article</title></head>
<body>
<div>
<p>This is a substantial article with enough content for readability
to detect it as readable. Multiple paragraphs of real text content
make the readability check more reliable. We continue writing to
ensure the character threshold is met by the detection algorithm.</p>
<p>Second paragraph adds more content density to help the heuristic
algorithms identify this as readable content. The check document
function requires sufficient paragraph text to return true.</p>
<p>Third paragraph is here to push us well past the minimum character
threshold. Readability looks for multiple paragraphs with substantial
text content before declaring a page readable.</p>
<p>Fourth paragraph for good measure. Having four or more paragraphs
each with reasonable length text tends to reliably trigger the
readability detection in most implementations.</p>
</div>
</body>
</html>`

	doc, err := htmlparser.Parse(strings.NewReader(articleHTML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if !IsReadable(doc) {
		t.Error("expected article page to be readable")
	}
}

func TestIsReadable_EmptyPage(t *testing.T) {
	const emptyHTML = `<!DOCTYPE html>
<html><head><title>Empty</title></head>
<body></body>
</html>`

	doc, err := htmlparser.Parse(strings.NewReader(emptyHTML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if IsReadable(doc) {
		t.Error("expected empty page to not be readable")
	}
}

func TestExtract_InvalidURL(t *testing.T) {
	const rawHTML = `<!DOCTYPE html><html><body><p>Test</p></body></html>`
	doc, err := htmlparser.Parse(strings.NewReader(rawHTML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, err = Extract(doc, "://invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
