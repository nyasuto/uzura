package markdown

import (
	"strings"
	"unicode/utf8"

	"github.com/nyasuto/uzura/internal/dom"
)

// spaIndicators are text patterns that suggest a page is a client-side rendered SPA.
// These are checked case-insensitively against the visible body text.
var spaIndicators = []string{
	"loading...",
	"please wait",
	"loading, please wait",
	"please enable javascript",
	"you need to enable javascript",
	"this app requires javascript",
}

// maxSPABodyLength is the maximum body text length (in runes) for a page
// to be considered a potential SPA. Pages with substantial content are not SPAs.
const maxSPABodyLength = 200

// DetectSPA checks whether a document appears to be a client-side rendered SPA
// that requires JavaScript execution to display content.
// It examines the visible body text for loading indicators and empty content.
func DetectSPA(doc *dom.Document) bool {
	body := doc.Body()
	if body == nil {
		return false
	}

	text := strings.TrimSpace(dom.CleanTextContent(body))

	// Empty body is a strong SPA signal
	if text == "" {
		return true
	}

	// If body text is very short, check for loading indicators
	runeCount := utf8.RuneCountInString(text)
	if runeCount > maxSPABodyLength {
		return false
	}

	lower := strings.ToLower(text)
	for _, indicator := range spaIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// DetectSPAFromContent checks whether extracted markdown content suggests
// the page is an SPA. This is used after readability extraction when the
// result quality is Partial or Failed.
func DetectSPAFromContent(content string, quality ContentQuality) bool {
	if quality == QualityGood {
		return false
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}

	lower := strings.ToLower(trimmed)
	for _, indicator := range spaIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}
