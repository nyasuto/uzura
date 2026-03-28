// Package page orchestrates DOM, network, and (future) JS execution
// for a single browsing page.
package page

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/html"
	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
)

var requestCounter atomic.Int64

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

// Page represents a single browsing page (tab).
// It coordinates fetching, parsing, and DOM construction.
type Page struct {
	fetcher            *network.Fetcher
	doc                *dom.Document
	url                string
	vm                 *js.VM
	vmOptions          []js.Option
	networkObserver    NetworkObserver
	requestInterceptor RequestInterceptor
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
	return &Page{fetcher: f, vmOptions: vmOpts, networkObserver: obs, requestInterceptor: intercept}
}

// Navigate loads the document at the given URL.
// It performs: robots check → fetch → decode → parse → DOM construction.
func (p *Page) Navigate(ctx context.Context, url string) error {
	reqID := fmt.Sprintf("req-%d", requestCounter.Add(1))
	now := func() float64 { return float64(time.Now().UnixMilli()) / 1000.0 }

	p.emit(NetworkEvent{
		Type:      NetworkRequestWillBeSent,
		RequestID: reqID,
		URL:       url,
		Method:    http.MethodGet,
		Timestamp: now(),
	})

	fetchURL := url
	var headerOverrides http.Header

	// Request interception hook: if an interceptor is set, pause and wait
	// for its decision before proceeding with the fetch.
	if p.requestInterceptor != nil {
		result, iErr := p.requestInterceptor(ctx, InterceptedRequest{
			RequestID: reqID,
			URL:       url,
			Method:    http.MethodGet,
			Stage:     StageRequest,
		})
		if iErr != nil {
			p.emitFailed(reqID, url, now(), iErr.Error())
			return fmt.Errorf("navigate %s: intercept: %w", url, iErr)
		}
		if result != nil {
			switch result.Action {
			case InterceptFail:
				reason := result.ErrorReason
				if reason == "" {
					reason = "BlockedByClient"
				}
				p.emitFailed(reqID, url, now(), reason)
				return fmt.Errorf("navigate %s: blocked by client (%s)", url, reason)
			case InterceptFulfill:
				return p.handleFulfill(reqID, url, now, result)
			case InterceptContinue:
				if result.URL != "" {
					fetchURL = result.URL
				}
				if result.Headers != nil {
					headerOverrides = result.Headers
				}
			}
		}
	}

	resp, err := p.fetchWithOverrides(ctx, fetchURL, headerOverrides)
	if err != nil {
		p.emitFailed(reqID, url, now(), err.Error())
		return fmt.Errorf("navigate %s: %w", url, err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	respHeaders := resp.Header
	respURL := resp.Request.URL.String()
	mimeType := mimeFromResponse(resp)

	reader, err := network.DecodeResponse(resp)
	if err != nil {
		return fmt.Errorf("navigate decode %s: %w", url, err)
	}
	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("navigate read %s: %w", url, err)
	}

	// Response-stage interception: pause after response is received.
	if p.requestInterceptor != nil {
		result, iErr := p.requestInterceptor(ctx, InterceptedRequest{
			RequestID:       reqID,
			URL:             fetchURL,
			Method:          http.MethodGet,
			Stage:           StageResponse,
			StatusCode:      statusCode,
			ResponseHeaders: respHeaders,
			Body:            bodyBytes,
		})
		if iErr != nil {
			p.emitFailed(reqID, url, now(), iErr.Error())
			return fmt.Errorf("navigate %s: response intercept: %w", url, iErr)
		}
		if result != nil {
			switch result.Action {
			case InterceptFail:
				reason := result.ErrorReason
				if reason == "" {
					reason = "BlockedByClient"
				}
				p.emitFailed(reqID, url, now(), reason)
				return fmt.Errorf("navigate %s: blocked by client (%s)", url, reason)
			case InterceptFulfill:
				return p.handleFulfill(reqID, url, now, result)
			case InterceptContinue:
				if result.RespStatusCode > 0 {
					statusCode = result.RespStatusCode
				}
				if result.RespHeaders != nil {
					for k, v := range result.RespHeaders {
						respHeaders.Set(k, v)
					}
				}
			}
		}
	}

	p.emit(NetworkEvent{
		Type:        NetworkResponseReceived,
		RequestID:   reqID,
		URL:         respURL,
		Timestamp:   now(),
		StatusCode:  statusCode,
		StatusText:  http.StatusText(statusCode),
		MimeType:    mimeType,
		RespHeaders: respHeaders,
	})

	p.emit(NetworkEvent{
		Type:              NetworkLoadingFinished,
		RequestID:         reqID,
		Timestamp:         now(),
		EncodedDataLength: int64(len(bodyBytes)),
		Body:              bodyBytes,
	})

	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("navigate parse %s: %w", url, err)
	}

	doc.SetQueryEngine(css.NewEngine())
	p.doc = doc
	p.url = url
	return nil
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
	return p.doc
}

// URL returns the URL of the currently loaded page.
func (p *Page) URL() string {
	return p.url
}

// VM returns the JavaScript VM, creating one if needed.
func (p *Page) VM() *js.VM {
	if p.vm == nil {
		p.vm = js.New(p.vmOptions...)
		if p.doc != nil {
			js.BindDocument(p.vm, p.doc)
		}
	}
	return p.vm
}
