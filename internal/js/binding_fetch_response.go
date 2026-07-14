package js

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dop251/goja"
)

// makeResponseObject builds the JS Response object returned by fetch's
// resolved promise. reqURL is the (already-resolved) request URL, used
// only to compute the `redirected` flag against resp.FinalURL.
//
// Not used by XMLHttpRequest, which exposes response data through its own
// properties/methods; this is fetch-internal, hence unexported.
func makeResponseObject(vm *VM, reqURL string, resp *HTTPResponse) *goja.Object {
	rt := vm.runtime
	obj := rt.NewObject()

	ok := resp.Status >= 200 && resp.Status < 300
	_ = obj.Set("ok", ok)
	_ = obj.Set("status", resp.Status)
	_ = obj.Set("statusText", resp.StatusText)
	_ = obj.Set("url", resp.FinalURL)
	_ = obj.Set("redirected", resp.FinalURL != "" && resp.FinalURL != reqURL)
	_ = obj.Set("headers", newHeadersObject(rt, resp.Headers))

	_ = obj.Set("text", func(goja.FunctionCall) goja.Value {
		p, resolve, _ := rt.NewPromise()
		_ = resolve(string(resp.Body))
		return rt.ToValue(p)
	})

	_ = obj.Set("json", func(goja.FunctionCall) goja.Value {
		p, resolve, reject := rt.NewPromise()
		var data interface{}
		if err := json.Unmarshal(resp.Body, &data); err != nil {
			_ = reject(rt.NewTypeError("fetch: response is not valid JSON: " + err.Error()))
			return rt.ToValue(p)
		}
		_ = resolve(rt.ToValue(data))
		return rt.ToValue(p)
	})

	_ = obj.Set("arrayBuffer", func(goja.FunctionCall) goja.Value {
		p, resolve, _ := rt.NewPromise()
		buf := make([]byte, len(resp.Body))
		copy(buf, resp.Body)
		_ = resolve(rt.NewArrayBuffer(buf))
		return rt.ToValue(p)
	})

	return obj
}

// newHeadersObject creates a standalone Headers-like JS object wrapping h.
func newHeadersObject(rt *goja.Runtime, h http.Header) *goja.Object {
	obj := rt.NewObject()
	populateHeadersMethods(rt, obj, h)
	return obj
}

// populateHeadersMethods installs get/has/forEach on obj, backed by h.
// get/has are case-insensitive (http.Header.Get/lookup canonicalizes the
// key); forEach visits every (value, name, headersObj) triple in the
// order net/http's map iteration yields them — no ordering guarantee, per
// the Headers spec.
func populateHeadersMethods(rt *goja.Runtime, obj *goja.Object, h http.Header) {
	_ = obj.Set("get", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		values := h.Values(name)
		if len(values) == 0 {
			return goja.Null()
		}
		return rt.ToValue(strings.Join(values, ", "))
	})
	_ = obj.Set("has", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		_, ok := h[http.CanonicalHeaderKey(name)]
		return rt.ToValue(ok)
	})
	_ = obj.Set("forEach", func(call goja.FunctionCall) goja.Value {
		cb, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return goja.Undefined()
		}
		for name, values := range h {
			lower := strings.ToLower(name)
			for _, v := range values {
				_, _ = cb(goja.Undefined(), rt.ToValue(v), rt.ToValue(lower), rt.ToValue(obj))
			}
		}
		return goja.Undefined()
	})
}
