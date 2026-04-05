package page

import (
	"context"

	"github.com/nyasuto/uzura/internal/dom"
)

// sendResourceHintsBg fires background requests for favicon.ico and
// pre-extracted sub-resource URLs (CSS/JS). Resource URLs must be extracted
// before calling this method to avoid racing with JS DOM mutations.
func (p *Page) sendResourceHintsBg(pageURL string, resourceURLs []string) {
	// Use a detached context so hints are not tied to the navigation context.
	ctx := context.Background()

	p.fetcher.FetchFavicon(ctx, pageURL)

	if len(resourceURLs) > 0 {
		p.fetcher.FetchResourceHints(ctx, pageURL, resourceURLs)
	}
}

// extractResourceURLs collects CSS and JS URLs from <link rel="stylesheet"> and
// <script src="..."> elements in the document.
func extractResourceURLs(doc *dom.Document) []string {
	var urls []string
	collectResourceURLs(doc, &urls)
	return urls
}

func collectResourceURLs(n dom.Node, urls *[]string) {
	if elem, ok := n.(*dom.Element); ok {
		switch elem.LocalName() {
		case "link":
			rel := elem.GetAttribute("rel")
			href := elem.GetAttribute("href")
			if rel == "stylesheet" && href != "" {
				*urls = append(*urls, href)
			}
		case "script":
			src := elem.GetAttribute("src")
			if src != "" {
				*urls = append(*urls, src)
			}
		}
	}
	for _, child := range n.ChildNodes() {
		collectResourceURLs(child, urls)
	}
}
