package markdown

import (
	"strings"
	"unicode/utf8"

	"github.com/nyasuto/uzura/internal/dom"
)

// minNoscriptLength is the minimum character count for a noscript block
// to be considered as useful content (not just "Enable JavaScript" messages).
const minNoscriptLength = 50

// ExtractNoscriptContent collects text content from all <noscript> elements
// in the document body. Noscript elements in <head> are skipped since they
// typically contain meta redirects rather than readable content.
func ExtractNoscriptContent(doc *dom.Document) []string {
	body := doc.Body()
	if body == nil {
		return nil
	}
	var contents []string
	collectNoscript(body, &contents)
	return contents
}

func collectNoscript(n dom.Node, out *[]string) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		el, ok := c.(*dom.Element)
		if !ok {
			continue
		}
		if el.LocalName() == "noscript" {
			text := strings.TrimSpace(el.TextContent())
			if text != "" {
				*out = append(*out, text)
			}
			continue
		}
		collectNoscript(c, out)
	}
}

// PickBestNoscript selects the best noscript content from a list of candidates.
// It returns the longest content that exceeds minNoscriptLength, or empty string
// if no candidate qualifies.
func PickBestNoscript(contents []string) string {
	best := ""
	bestLen := 0
	for _, c := range contents {
		n := utf8.RuneCountInString(c)
		if n >= minNoscriptLength && n > bestLen {
			best = c
			bestLen = n
		}
	}
	return best
}

// ShouldUseNoscript determines whether noscript content should be used
// instead of the normal extraction result.
// Noscript is used only when:
//  1. Normal extraction quality is Partial or Failed, AND
//  2. The noscript content itself is substantial (>= minContentLength characters).
func ShouldUseNoscript(normalQuality ContentQuality, noscriptContent string) bool {
	if normalQuality == QualityGood {
		return false
	}
	return AssessQuality(noscriptContent) == QualityGood
}
