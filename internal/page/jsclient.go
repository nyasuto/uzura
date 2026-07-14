package page

import (
	"context"
	"io"
	"net/http"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
)

// maxJSResponseBytes caps a single JS-initiated response body (10 MB).
const maxJSResponseBytes = 10 << 20

// jsHTTPClient adapts the page's Fetcher for JS-initiated requests
// (fetch / XMLHttpRequest). Cookie jar, UA, compression handling and
// robots policy are shared with page navigation. No CORS checks are
// applied: uzura is an agent browser and does not carry a human user's
// cross-site credentials.
func (p *Page) jsHTTPClient() js.HTTPClient {
	return func(ctx context.Context, req js.HTTPRequest) (*js.HTTPResponse, error) {
		headers := http.Header{}
		for k, v := range req.Headers {
			headers.Set(k, v)
		}
		resp, err := p.fetcher.FetchRequest(ctx, req.Method, req.URL, headers, req.Body)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if decompErr := network.DecompressResponse(resp); decompErr != nil {
			return nil, decompErr
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSResponseBytes))
		if err != nil {
			return nil, err
		}
		finalURL := req.URL
		if resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		return &js.HTTPResponse{
			Status:     resp.StatusCode,
			StatusText: http.StatusText(resp.StatusCode),
			Headers:    resp.Header,
			Body:       body,
			FinalURL:   finalURL,
		}, nil
	}
}

// bindAll wires every JS web-API binding onto vm against doc/pageURL, using
// the page's HTTP client and its (persisted-across-navigations) storage
// backends. Shared by VM() (lazy access without running scripts) and
// runScripts (post-navigation script execution).
func (p *Page) bindAll(vm *js.VM, doc *dom.Document, pageURL string, local, session js.Storage) {
	js.BindDocument(vm, doc)
	vm.SetHTTPClient(p.jsHTTPClient())
	js.BindFetch(vm)
	js.BindAbort(vm)
	js.BindXHR(vm)
	js.BindStorage(vm, local, session)
	js.BindLocation(vm, pageURL)
}

// runScripts executes the document's inline scripts and drives the event
// loop until all JS-initiated work (fetch/XHR/timers) settles or ctx is
// done. Script errors and event-loop ctx errors never fail navigation
// (partial-result policy): they are ignored here, and Navigate still
// returns nil for them.
//
// The VM is built and run on a local variable and is only published to
// p.vm once script execution and the event loop have fully settled.
// goja.Runtime is not safe for concurrent use, so the in-progress runtime
// must stay private to this goroutine until it's done: a concurrent VM()
// call (e.g. CDP Runtime.evaluate/callFunctionOn) during that window sees
// the still-nil (or stale, previous-navigation) p.vm and builds its own
// separate VM instead of reaching into the one being executed here. That
// VM() caller briefly sees an out-of-date state (or triggers an extra VM
// build), but it can never race with runScripts, because the two never
// share the same *goja.Runtime pointer while either might be running JS
// on it.
func (p *Page) runScripts(ctx context.Context) {
	p.mu.Lock()
	doc := p.doc
	pageURL := p.url
	if p.localStore == nil {
		p.localStore = js.NewMemStorage()
		p.sessionStore = js.NewMemStorage()
	}
	local, session := p.localStore, p.sessionStore
	p.mu.Unlock()

	vm := js.New(p.vmOptions...)
	if doc == nil {
		// Nothing to execute; still refresh p.vm as before (a fresh,
		// unbound VM), matching prior behavior for this edge case.
		p.mu.Lock()
		p.vm = vm
		p.mu.Unlock()
		return
	}

	p.bindAll(vm, doc, pageURL, local, session)

	// Prime the loop's context BEFORE executing top-level scripts: a
	// fetch()/XMLHttpRequest call made directly from a script's top-level
	// body (as opposed to from inside a setTimeout/task callback that only
	// runs once RunEventLoopContext is already pumping) reads vm.LoopContext()
	// synchronously, before RunEventLoopContext below would otherwise have
	// set it. Without this, that call captures context.Background() instead
	// of ctx: the page deadline can never cancel its derived request context,
	// leaking both the goroutine and the context until the request happens
	// to finish on its own. RunEventLoopContext re-sets the identical ctx and
	// its deferred clear still leaves vm.loop's context cleanly reset to nil
	// once everything settles, so this priming leaves no stale state behind.
	vm.SetLoopContext(ctx)
	_ = js.ExecuteScripts(vm, doc)
	_ = vm.RunEventLoopContext(ctx)

	p.mu.Lock()
	p.vm = vm
	p.mu.Unlock()
}
