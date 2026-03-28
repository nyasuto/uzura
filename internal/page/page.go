// Package page orchestrates DOM, network, and (future) JS execution
// for a single browsing page.
package page

import (
	"context"
	"fmt"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/network"
)

// Page represents a single browsing page (tab).
// It coordinates fetching, parsing, and DOM construction.
type Page struct {
	fetcher *network.Fetcher
	doc     *dom.Document
	url     string
}

// Options configures a Page.
type Options struct {
	Fetcher *network.Fetcher
}

// New creates a new Page with the given options.
// If opts is nil or opts.Fetcher is nil, a default Fetcher is created.
func New(opts *Options) *Page {
	var f *network.Fetcher
	if opts != nil && opts.Fetcher != nil {
		f = opts.Fetcher
	} else {
		f = network.NewFetcher(nil)
	}
	return &Page{fetcher: f}
}

// Navigate loads the document at the given URL.
// It performs: robots check → fetch → decode → parse → DOM construction.
func (p *Page) Navigate(ctx context.Context, url string) error {
	resp, err := p.fetcher.FetchContext(ctx, url)
	if err != nil {
		return fmt.Errorf("navigate %s: %w", url, err)
	}
	defer resp.Body.Close()

	reader, err := network.DecodeResponse(resp)
	if err != nil {
		return fmt.Errorf("navigate decode %s: %w", url, err)
	}

	doc, err := html.Parse(reader)
	if err != nil {
		return fmt.Errorf("navigate parse %s: %w", url, err)
	}

	doc.SetQueryEngine(css.NewEngine())
	p.doc = doc
	p.url = url
	return nil
}

// Document returns the current DOM Document, or nil if no page is loaded.
func (p *Page) Document() *dom.Document {
	return p.doc
}

// URL returns the URL of the currently loaded page.
func (p *Page) URL() string {
	return p.url
}
