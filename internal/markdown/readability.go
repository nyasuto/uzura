// Package markdown provides DOM-to-Markdown conversion with readability extraction.
package markdown

import (
	"io"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"

	"github.com/nyasuto/uzura/internal/dom"
)

// ExtractResult holds the result of readability extraction.
type ExtractResult struct {
	Title   string
	Byline  string
	Excerpt string
	Content string // cleaned HTML of the main content
}

// Extract runs readability on a DOM document and returns the extracted article.
// If extraction fails (e.g., non-article page), it returns an error.
func Extract(doc *dom.Document, pageURL string) (*ExtractResult, error) {
	serialized := dom.Serialize(doc)
	r := strings.NewReader(serialized)

	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, err
	}

	article, err := readability.FromReader(r, u)
	if err != nil {
		return nil, err
	}

	var content strings.Builder
	if article.Node != nil {
		if err := article.RenderHTML(&content); err != nil {
			return nil, err
		}
	}

	return &ExtractResult{
		Title:   article.Title(),
		Byline:  article.Byline(),
		Excerpt: article.Excerpt(),
		Content: content.String(),
	}, nil
}

// IsReadable checks whether a document likely contains article content
// without performing full extraction.
func IsReadable(doc *dom.Document) bool {
	serialized := dom.Serialize(doc)
	r := strings.NewReader(serialized)

	parser := readability.NewParser()
	parsed, err := parseHTMLNode(r)
	if err != nil {
		return false
	}
	return parser.CheckDocument(parsed)
}

func parseHTMLNode(r io.Reader) (*html.Node, error) {
	return html.Parse(r)
}
