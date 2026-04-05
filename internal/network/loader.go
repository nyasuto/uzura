package network

import (
	"context"
	"fmt"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/html"
)

// LoadDocument fetches a URL, decodes the response body to UTF-8,
// parses the HTML, and returns a DOM Document.
func (f *Fetcher) LoadDocument(url string) (*dom.Document, error) {
	return f.LoadDocumentContext(context.Background(), url)
}

// LoadDocumentContext fetches a URL with context support, decodes the
// response body to UTF-8, parses the HTML, and returns a DOM Document.
func (f *Fetcher) LoadDocumentContext(ctx context.Context, url string) (*dom.Document, error) {
	if f.obeyRobots {
		if err := f.checkRobots(url); err != nil {
			return nil, err
		}
	}

	resp, err := f.FetchContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if decompErr := DecompressResponse(resp); decompErr != nil {
		return nil, fmt.Errorf("decompressing %s: %w", url, decompErr)
	}

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
