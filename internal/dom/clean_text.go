package dom

import "strings"

// skipTextTags are elements whose text content should be excluded
// from clean text extraction (script, style, noscript).
var skipTextTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
}

// CleanTextContent extracts visible text content from a node,
// skipping script, style, and noscript elements.
// Unlike TextContent(), this is optimized for AI-readable output.
func CleanTextContent(n Node) string {
	var sb strings.Builder
	collectCleanText(&sb, n)
	return sb.String()
}

func collectCleanText(sb *strings.Builder, n Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *Text:
			sb.WriteString(v.Data)
		case *Element:
			if !skipTextTags[v.LocalName()] {
				collectCleanText(sb, v)
			}
		}
	}
}
