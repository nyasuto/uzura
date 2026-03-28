package browser

import (
	"errors"
	"net/http/cookiejar"
	"sync"

	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

// BrowserContext provides session isolation: each context has its own
// cookie jar and set of pages. Contexts created by Browser.NewContext()
// do not share cookies or state with each other.
type BrowserContext struct {
	mu        sync.Mutex
	browser   *Browser
	fetcher   *network.Fetcher
	pages     []*page.Page
	isDefault bool
	closed    bool
}

func newBrowserContext(b *Browser, isDefault bool) *BrowserContext {
	jar, _ := cookiejar.New(nil)

	fetchOpts := &network.FetcherOptions{
		CookieJar: jar,
	}
	if b.opts.UserAgent != "" {
		fetchOpts.UserAgent = b.opts.UserAgent
	}
	if b.opts.Timeout > 0 {
		fetchOpts.Timeout = b.opts.Timeout
	}

	return &BrowserContext{
		browser:   b,
		fetcher:   network.NewFetcher(fetchOpts),
		isDefault: isDefault,
	}
}

// NewPage creates a new Page within this context.
// The page shares the context's cookie jar.
func (c *BrowserContext) NewPage() (*page.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errors.New("browser context is closed")
	}

	// Check browser-level closed state.
	c.browser.mu.Lock()
	browserClosed := c.browser.closed
	c.browser.mu.Unlock()
	if browserClosed {
		return nil, ErrBrowserClosed
	}

	// Check MaxPages limit.
	if !c.browser.acquirePage() {
		return nil, ErrMaxPagesReached
	}

	p := page.New(&page.Options{
		Fetcher: c.fetcher,
	})
	p.SetCloseObserver(func(_ *page.Page) {
		c.browser.releasePage()
		c.removePage(p)
	})
	c.pages = append(c.pages, p)
	return p, nil
}

func (c *BrowserContext) removePage(p *page.Page) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, pg := range c.pages {
		if pg == p {
			c.pages = append(c.pages[:i], c.pages[i+1:]...)
			return
		}
	}
}

// Pages returns all active pages in this context.
func (c *BrowserContext) Pages() []*page.Page {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]*page.Page, len(c.pages))
	copy(result, c.pages)
	return result
}

// Close closes this context and all its pages.
// The default context cannot be closed; attempting to do so returns an error.
func (c *BrowserContext) Close() error {
	if c.isDefault {
		return errors.New("cannot close the default browser context")
	}

	c.closeInternal()
	c.browser.removeContext(c)
	return nil
}

func (c *BrowserContext) closeInternal() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	pages := make([]*page.Page, len(c.pages))
	copy(pages, c.pages)
	c.pages = nil
	c.mu.Unlock()

	for _, p := range pages {
		_ = p.Close()
	}
}

// Browser returns the parent Browser.
func (c *BrowserContext) Browser() *Browser {
	return c.browser
}
