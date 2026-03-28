package network

import (
	"fmt"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/html"
)

// LoadDocument fetches a URL, decodes the response body to UTF-8,
// parses the HTML, and returns a DOM Document.
func (f *Fetcher) LoadDocument(url string) (*dom.Document, error) {
	resp, err := f.Fetch(url)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", url, err)
	}
	defer resp.Body.Close()

	reader, err := DecodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", url, err)
	}

	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", url, err)
	}

	return doc, nil
}
