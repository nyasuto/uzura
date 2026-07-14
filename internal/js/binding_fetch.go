package js

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dop251/goja"
)

// BindFetch registers the fetch() function and the Headers class on the
// VM's global object. Callers (page setup, tests) invoke this explicitly
// rather than it running from an implicit init(), so standalone VMs that
// never touch the network stay free of it.
//
// No CORS / same-origin checks are applied: uzura is an agent browser and
// does not carry a human user's ambient credentials across sites (see
// HTTPClient in http.go).
func BindFetch(vm *VM) {
	_ = vm.runtime.Set("fetch", vm.jsFetch)
	registerHeadersClass(vm)
}

// jsFetch implements the global fetch(url, init) function. It always
// resolves with a Response object for HTTP-level outcomes (including 4xx
// and 5xx) and only rejects for transport-level failures (DNS, connection
// refused, etc. — whatever the injected HTTPClient reports as an error).
func (vm *VM) jsFetch(call goja.FunctionCall) goja.Value {
	rt := vm.runtime
	promise, resolve, reject := rt.NewPromise()

	req, err := parseFetchArgs(vm, call)
	if err != nil {
		_ = reject(rt.NewTypeError(err.Error()))
		return rt.ToValue(promise)
	}

	client := vm.httpClient()
	ctx := vm.LoopContext()
	vm.loop.addPending()
	go func() {
		var resp *HTTPResponse
		var err error
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					err = fmt.Errorf("fetch: %v", rec)
				}
			}()
			resp, err = client(ctx, req)
		}()
		// A single completeWith call per goroutine invocation keeps the
		// addPending/pending-- balance exact even on the panic path above.
		vm.loop.completeWith(func() {
			if err != nil {
				_ = reject(rt.NewTypeError("fetch failed: " + err.Error()))
				return
			}
			_ = resolve(makeResponseObject(vm, req.URL, resp))
		})
	}()

	return rt.ToValue(promise)
}

// parseFetchArgs builds an HTTPRequest from fetch's (url, init) arguments.
// url is resolved against the document's base URL. init.method defaults to
// GET and is upper-cased; init.headers copies every own key of the given
// object; init.body (if present) is coerced to a string. Unknown options
// (credentials, mode, signal, ...) are silently ignored — uzura has no
// same-origin/CORS model and no request cancellation wiring yet.
func parseFetchArgs(vm *VM, call goja.FunctionCall) (HTTPRequest, error) {
	rt := vm.runtime
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Argument(0)) {
		return HTTPRequest{}, fmt.Errorf("fetch: 1 argument required, url is missing")
	}

	req := HTTPRequest{
		Method:  http.MethodGet,
		URL:     vm.resolveURL(call.Argument(0).String()),
		Headers: map[string]string{},
	}

	initArg := call.Argument(1)
	if goja.IsUndefined(initArg) || goja.IsNull(initArg) {
		return req, nil
	}
	init := initArg.ToObject(rt)

	if m := init.Get("method"); m != nil && !goja.IsUndefined(m) {
		req.Method = strings.ToUpper(m.String())
	}
	if h := init.Get("headers"); h != nil && !goja.IsUndefined(h) && !goja.IsNull(h) {
		hObj := h.ToObject(rt)
		for _, key := range hObj.Keys() {
			req.Headers[key] = hObj.Get(key).String()
		}
	}
	if b := init.Get("body"); b != nil && !goja.IsUndefined(b) && !goja.IsNull(b) {
		req.Body = []byte(b.String())
	}

	return req, nil
}

// registerHeadersClass registers the Headers constructor. It exists mainly
// so `new Headers(...)` and `typeof Headers === 'function'` feature
// detection work from JS; fetch's own init.headers parsing (above) accepts
// a plain object directly and does not require a Headers instance.
func registerHeadersClass(vm *VM) {
	rt := vm.runtime
	_ = rt.Set("Headers", func(call goja.ConstructorCall) *goja.Object {
		h := make(http.Header)
		if len(call.Arguments) > 0 {
			initArg := call.Argument(0)
			if !goja.IsUndefined(initArg) && !goja.IsNull(initArg) {
				obj := initArg.ToObject(rt)
				for _, key := range obj.Keys() {
					h.Set(key, obj.Get(key).String())
				}
			}
		}
		populateHeadersMethods(rt, call.This, h)
		return call.This
	})
}
