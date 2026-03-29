package markdown

import (
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
)

// stripTags are elements always removed regardless of mode.
var stripTags = map[string]bool{
	"script": true, "style": true, "noscript": true,
}

// layoutTags are elements removed only in article extraction mode.
var layoutTags = map[string]bool{
	"nav": true, "header": true, "footer": true, "aside": true,
}

// adClassPatterns are substrings in class/id that indicate ad content.
var adClassPatterns = []string{
	"ad-", "ad_", "ads-", "ads_", "advert",
	"sidebar", "promo", "sponsor", "banner-ad",
}

// Clean removes unwanted elements from a DOM tree.
// If articleMode is true, layout elements (nav, header, footer, aside) are also removed.
// The function modifies the tree in place.
func Clean(doc *dom.Document, articleMode bool) {
	cleanNode(doc, articleMode)
}

func cleanNode(n dom.Node, articleMode bool) {
	var toRemove []dom.Node

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		el, ok := child.(*dom.Element)
		if !ok {
			continue
		}

		tag := el.LocalName()

		if stripTags[tag] {
			toRemove = append(toRemove, child)
			continue
		}

		if articleMode && layoutTags[tag] {
			toRemove = append(toRemove, child)
			continue
		}

		if isHidden(el) {
			toRemove = append(toRemove, child)
			continue
		}

		if isAdElement(el) {
			toRemove = append(toRemove, child)
			continue
		}

		// Recurse into surviving children
		cleanNode(child, articleMode)
	}

	for _, child := range toRemove {
		n.RemoveChild(child)
	}
}

func isHidden(el *dom.Element) bool {
	if el.HasAttribute("hidden") {
		return true
	}
	if el.GetAttribute("aria-hidden") == "true" {
		return true
	}
	style := el.GetAttribute("style")
	if style != "" && strings.Contains(strings.ToLower(style), "display:none") {
		return true
	}
	if style != "" && strings.Contains(strings.ToLower(style), "display: none") {
		return true
	}
	return false
}

func isAdElement(el *dom.Element) bool {
	cls := strings.ToLower(el.GetAttribute("class"))
	id := strings.ToLower(el.GetAttribute("id"))
	combined := cls + " " + id
	for _, pattern := range adClassPatterns {
		if strings.Contains(combined, pattern) {
			return true
		}
	}
	return false
}
