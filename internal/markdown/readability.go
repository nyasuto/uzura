// Package markdown provides DOM-to-Markdown conversion with readability extraction.
package markdown

import (
	"fmt"
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

// inlinePhrasingTags are inline elements whose class and id attributes are
// stripped before readability extraction. Readability's unlikely-candidate
// heuristic matches noise patterns ("extra", "header", "sidebar", ...)
// against class+id of every element, which false-positives on inline
// phrasing content (issue #31: Nextra's class="nextra-code" matches "extra").
// Readability strips classes from its output anyway (KeepClasses=false),
// so removing them up front does not change the extracted result.
var inlinePhrasingTags = map[string]bool{
	"code": true, "em": true, "strong": true, "span": true,
	"b": true, "i": true, "u": true, "s": true, "small": true,
	"kbd": true, "samp": true, "var": true, "abbr": true,
	"mark": true, "sub": true, "sup": true, "time": true,
	"cite": true, "q": true, "dfn": true, "data": true,
}

// Extract runs readability on a DOM document and returns the extracted article.
// If extraction fails (e.g., non-article page), it returns an error.
// It recovers from panics in the readability library for malformed documents.
func Extract(doc *dom.Document, pageURL string) (result *ExtractResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("readability panic: %v", r)
		}
	}()

	if cloned, ok := doc.CloneNode(true).(*dom.Document); ok {
		stripInlineAttributes(cloned)
		doc = cloned
	}

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

// stripInlineAttributes removes class and id attributes from inline phrasing
// elements throughout the tree. See inlinePhrasingTags for why.
func stripInlineAttributes(n dom.Node) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if el, ok := child.(*dom.Element); ok && inlinePhrasingTags[el.LocalName()] {
			el.RemoveAttribute("class")
			el.RemoveAttribute("id")
		}
		stripInlineAttributes(child)
	}
}
