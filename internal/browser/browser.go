// Package browser provides the top-level Browser and BrowserContext
// abstractions for managing multiple pages with session isolation.
package browser

import (
	"sync"
	"time"
)

// Browser represents the top-level browser instance.
// It manages BrowserContexts, each of which isolates cookies and pages.
type Browser struct {
	mu         sync.Mutex
	contexts   []*BrowserContext
	defaultCtx *BrowserContext
	opts       Options
	closed     bool
}

// Options configures a Browser.
type Options struct {
	UserAgent string
	Timeout   time.Duration
}

// Option is a functional option for New.
type Option func(*Options)

// WithUserAgent sets the User-Agent for all fetchers created by this browser.
func WithUserAgent(ua string) Option {
	return func(o *Options) { o.UserAgent = ua }
}

// WithTimeout sets the default request timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) { o.Timeout = d }
}

// New creates a new Browser with the given options.
// A default BrowserContext is created automatically.
func New(opts ...Option) *Browser {
	var o Options
	for _, fn := range opts {
		fn(&o)
	}

	b := &Browser{opts: o}
	b.defaultCtx = b.newContext(true)
	return b
}

// DefaultContext returns the default BrowserContext.
func (b *Browser) DefaultContext() *BrowserContext {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.defaultCtx
}

// NewContext creates a new isolated BrowserContext with its own cookie jar.
func (b *Browser) NewContext() *BrowserContext {
	return b.newContext(false)
}

func (b *Browser) newContext(isDefault bool) *BrowserContext {
	b.mu.Lock()
	defer b.mu.Unlock()

	ctx := newBrowserContext(b, isDefault)
	b.contexts = append(b.contexts, ctx)
	return ctx
}

// Contexts returns all active BrowserContexts (including the default).
func (b *Browser) Contexts() []*BrowserContext {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]*BrowserContext, len(b.contexts))
	copy(result, b.contexts)
	return result
}

func (b *Browser) removeContext(ctx *BrowserContext) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, c := range b.contexts {
		if c == ctx {
			b.contexts = append(b.contexts[:i], b.contexts[i+1:]...)
			return
		}
	}
}

// Close shuts down the browser, closing all contexts and their pages.
func (b *Browser) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	contexts := make([]*BrowserContext, len(b.contexts))
	copy(contexts, b.contexts)
	b.contexts = nil
	b.mu.Unlock()

	for _, ctx := range contexts {
		ctx.closeInternal()
	}
	return nil
}
