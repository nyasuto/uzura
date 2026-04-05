package markdown

import (
	"io"
	"strings"
	"unicode/utf8"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

// minContentLength is the minimum character count for readability output
// to be considered successful. Below this threshold, the content is likely
// incomplete (e.g., "Loading...", empty placeholder).
const minContentLength = 100

// ContentQuality describes the quality of extracted content.
type ContentQuality int

const (
	// QualityGood means readability extraction succeeded with sufficient content.
	QualityGood ContentQuality = iota
	// QualityPartial means some content was extracted but it's suspiciously short.
	QualityPartial
	// QualityFailed means readability extraction failed or returned empty content.
	QualityFailed
)

// AssessQuality evaluates the quality of readability-extracted content.
func AssessQuality(content string) ContentQuality {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return QualityFailed
	}
	charCount := utf8.RuneCountInString(trimmed)
	if charCount < minContentLength {
		return QualityPartial
	}
	return QualityGood
}

// contentRegionTags defines the priority order for content region fallback.
var contentRegionTags = []string{"main", "article"}

// FindContentRegion searches for the best content region in a document.
// It tries <main>, then <article>, then falls back to <body>.
// Returns the element to use as the content root.
func FindContentRegion(doc *dom.Document) dom.Node {
	for _, tag := range contentRegionTags {
		elems := doc.GetElementsByTagName(tag)
		if len(elems) > 0 {
			// If multiple matches, pick the one with the most text content
			return pickLargest(elems)
		}
	}
	// Fall back to body
	if body := doc.Body(); body != nil {
		return body
	}
	return doc.DocumentElement()
}

// pickLargest returns the element with the longest text content.
func pickLargest(elems []*dom.Element) *dom.Element {
	best := elems[0]
	bestLen := utf8.RuneCountInString(best.TextContent())
	for _, el := range elems[1:] {
		n := utf8.RuneCountInString(el.TextContent())
		if n > bestLen {
			best = el
			bestLen = n
		}
	}
	return best
}

// RenderWithFallback produces markdown from a document using a multi-stage
// extraction strategy:
//  1. Try readability extraction — if quality is Good, use it.
//  2. If quality is Partial or Failed, find the best content region
//     (<main> → <article> → <body>) and convert that.
func RenderWithFallback(doc *dom.Document, pageURL string) string {
	meta := ExtractMetadata(doc, pageURL)

	// Stage 1: Try readability extraction
	result, err := Extract(doc, pageURL)
	if err == nil {
		quality := AssessQuality(result.Content)
		if quality == QualityGood {
			md := convertExtractedHTML(result.Content)
			if md != "" {
				return FormatFrontmatter(meta) + "\n" + md
			}
		}
	}

	// Stage 2: Find best content region and convert
	cloned, ok := doc.CloneNode(true).(*dom.Document)
	if !ok {
		return FormatFrontmatter(meta) + "\n" + doc.DocumentElement().TextContent()
	}

	Clean(cloned, false) // don't strip layout tags yet — we need them for region detection
	region := FindContentRegion(cloned)

	// Clean layout elements within the chosen region
	cleanLayoutInRegion(region)

	body := Convert(region)
	return FormatFrontmatter(meta) + "\n" + body
}

// convertExtractedHTML parses readability-extracted HTML and converts to markdown.
func convertExtractedHTML(content string) string {
	r := strings.NewReader("<html><body>" + content + "</body></html>")
	doc, err := parseHTMLDoc(r)
	if err != nil {
		return ""
	}
	Clean(doc, true)
	return Convert(doc)
}

func parseHTMLDoc(r io.Reader) (*dom.Document, error) {
	return htmlparser.Parse(r)
}

// cleanLayoutInRegion removes nav/header/footer/aside from within a content region.
func cleanLayoutInRegion(n dom.Node) {
	var toRemove []dom.Node
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		el, ok := child.(*dom.Element)
		if !ok {
			continue
		}
		if layoutTags[el.LocalName()] {
			toRemove = append(toRemove, child)
			continue
		}
		cleanLayoutInRegion(child)
	}
	for _, child := range toRemove {
		n.RemoveChild(child)
	}
}
