// Package page orchestrates DOM, network, and (future) JS execution
// for a single browsing page.
package page

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
)

var (
	requestCounter atomic.Int64
	pageIDCounter  atomic.Int64
)

// ErrPageClosed is returned when an operation is attempted on a closed page.
var ErrPageClosed = errors.New("page is closed")

// NetworkEvent represents a network event emitted during page loading.
type NetworkEvent struct {
	Type      NetworkEventType
	RequestID string
	URL       string
	Method    string
	Headers   http.Header
	Timestamp float64

	// Response fields (populated for ResponseReceived, LoadingFinished).
	StatusCode  int
	StatusText  string
	MimeType    string
	RespHeaders http.Header

	// LoadingFinished fields.
	EncodedDataLength int64

	// LoadingFailed fields.
	ErrorText string

	// Body stored for getResponseBody.
	Body   []byte
	Base64 bool
}

// NetworkEventType identifies the kind of network event.
type NetworkEventType int

const (
	// NetworkRequestWillBeSent fires before the HTTP request.
	NetworkRequestWillBeSent NetworkEventType = iota
	// NetworkResponseReceived fires when response headers arrive.
	NetworkResponseReceived
	// NetworkLoadingFinished fires when the response body is fully read.
	NetworkLoadingFinished
	// NetworkLoadingFailed fires on a fetch error.
	NetworkLoadingFailed
)

// NetworkObserver is called for each network event during navigation.
type NetworkObserver func(evt NetworkEvent)

// CloseObserver is called when a page is closed.
type CloseObserver func(p *Page)

// Page represents a single browsing page (tab).
// It coordinates fetching, parsing, and DOM construction.
type Page struct {
	mu                 sync.Mutex
	id                 string
	fetcher            *network.Fetcher
	doc                *dom.Document
	url                string
	vm                 *js.VM
	vmOptions          []js.Option
	networkObserver    NetworkObserver
	requestInterceptor RequestInterceptor
	closeObserver      CloseObserver
	closed             bool
}

// Options configures a Page.
type Options struct {
	Fetcher            *network.Fetcher
	VMOptions          []js.Option
	NetworkObserver    NetworkObserver
	RequestInterceptor RequestInterceptor
}

// New creates a new Page with the given options.
// If opts is nil or opts.Fetcher is nil, a default Fetcher is created.
func New(opts *Options) *Page {
	var f *network.Fetcher
	var vmOpts []js.Option
	var obs NetworkObserver
	var intercept RequestInterceptor
	if opts != nil {
		if opts.Fetcher != nil {
			f = opts.Fetcher
		}
		vmOpts = opts.VMOptions
		obs = opts.NetworkObserver
		intercept = opts.RequestInterceptor
	}
	if f == nil {
		f = network.NewFetcher(nil)
	}
	id := fmt.Sprintf("page-%d", pageIDCounter.Add(1))
	return &Page{
		id:                 id,
		fetcher:            f,
		vmOptions:          vmOpts,
		networkObserver:    obs,
		requestInterceptor: intercept,
	}
}

// ID returns the unique identifier for this page.
func (p *Page) ID() string {
	return p.id
}

// SetCloseObserver sets a callback invoked when this page is closed.
func (p *Page) SetCloseObserver(obs CloseObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closeObserver = obs
}

// Close releases all resources held by this page (VM, DOM, observers).
// After Close, the page must not be used.
func (p *Page) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	obs := p.closeObserver
	p.vm = nil
	p.doc = nil
	p.url = ""
	p.networkObserver = nil
	p.requestInterceptor = nil
	p.closeObserver = nil
	p.mu.Unlock()

	if obs != nil {
		obs(p)
	}
	return nil
}

// IsClosed returns whether this page has been closed.
func (p *Page) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

func (p *Page) emit(evt NetworkEvent) {
	if p.networkObserver != nil {
		p.networkObserver(evt)
	}
}

// SetNetworkObserver sets the network observer callback.
func (p *Page) SetNetworkObserver(obs NetworkObserver) {
	p.networkObserver = obs
}

// SetRequestInterceptor sets or clears the request interceptor.
// When set, every request is passed to the interceptor before fetching.
func (p *Page) SetRequestInterceptor(i RequestInterceptor) {
	p.requestInterceptor = i
}

// Document returns the current DOM Document, or nil if no page is loaded.
func (p *Page) Document() *dom.Document {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.doc
}

// URL returns the URL of the currently loaded page.
func (p *Page) URL() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.url
}

// VM returns the JavaScript VM, creating one if needed.
func (p *Page) VM() *js.VM {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.vm == nil {
		p.vm = js.New(p.vmOptions...)
		if p.doc != nil {
			js.BindDocument(p.vm, p.doc)
		}
	}
	return p.vm
}
