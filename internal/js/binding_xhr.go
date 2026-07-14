package js

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// XMLHttpRequest readyState constants, mirrored as properties on both the
// constructor function and every instance (see BindXHR / newXHRObject).
const (
	xhrUnsent          = 0
	xhrOpened          = 1
	xhrHeadersReceived = 2
	xhrLoading         = 3
	xhrDone            = 4
)

// xhrState holds the Go-side state backing one XMLHttpRequest instance. It
// is only ever mutated from the event-loop thread: open/setRequestHeader/
// send are synchronous JS calls, and the response is delivered back via
// vm.loop.completeWith (also loop-thread). The in-flight goroutine spawned
// by send() only calls the injected HTTPClient and completeWith — it never
// touches xhrState directly.
type xhrState struct {
	method  string
	url     string
	headers map[string]string
	async   bool
	timeout time.Duration

	readyState int
	status     int
	statusText string
	respHeader http.Header
	body       []byte

	responseType string

	onReadyStateChange goja.Callable
	onLoad             goja.Callable
	onError            goja.Callable
	onTimeout          goja.Callable
	onAbort            goja.Callable
	listeners          map[string][]goja.Callable

	cancel context.CancelFunc

	// timeoutCtx is the context created from state.timeout by xhrSend (nil
	// when no timeout is set). deliverXHRResult uses it, rather than the
	// error's mere DeadlineExceeded-ness, to decide between firing ontimeout
	// and onerror: a page-level deadline threaded through
	// RunEventLoopContext also produces context.DeadlineExceeded, but must
	// not be misreported as this XHR's own timeout.
	timeoutCtx context.Context
}

// BindXHR registers the global XMLHttpRequest constructor on vm, including
// its readyState constants (both on the constructor and on each instance,
// matching the browser XMLHttpRequest.UNSENT-style static/instance
// duplication). Callers must also have configured an HTTPClient
// (vm.SetHTTPClient) and, typically, a base URL (vm.SetBaseURL) for
// relative request URLs to resolve.
func BindXHR(vm *VM) {
	rt := vm.runtime

	ctor := func(call goja.ConstructorCall) *goja.Object {
		state := &xhrState{
			headers:   map[string]string{},
			listeners: map[string][]goja.Callable{},
		}
		vm.buildXHRObject(call.This, state)
		return call.This
	}

	ctorVal := rt.ToValue(ctor)
	ctorObj := ctorVal.ToObject(rt)
	setReadyStateConstants(rt, ctorObj)
	_ = rt.Set("XMLHttpRequest", ctorVal)
}

// setReadyStateConstants sets UNSENT/OPENED/HEADERS_RECEIVED/LOADING/DONE
// on obj (used for both the constructor and each instance).
func setReadyStateConstants(rt *goja.Runtime, obj *goja.Object) {
	_ = obj.Set("UNSENT", xhrUnsent)
	_ = obj.Set("OPENED", xhrOpened)
	_ = obj.Set("HEADERS_RECEIVED", xhrHeadersReceived)
	_ = obj.Set("LOADING", xhrLoading)
	_ = obj.Set("DONE", xhrDone)
}

// buildXHRObject installs every XMLHttpRequest property and method on obj,
// backed by state.
func (vm *VM) buildXHRObject(obj *goja.Object, state *xhrState) {
	rt := vm.runtime
	setReadyStateConstants(rt, obj)

	vm.setXHRAccessors(obj, state)

	_ = obj.Set("open", func(call goja.FunctionCall) goja.Value {
		method := call.Argument(0).String()
		url := call.Argument(1).String()
		async := true
		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Argument(2)) {
			async = call.Argument(2).ToBoolean()
		}
		if !async {
			panic(rt.NewTypeError("XMLHttpRequest: synchronous requests are not supported"))
		}
		state.method = strings.ToUpper(method)
		state.url = vm.resolveURL(url)
		state.async = async
		state.headers = map[string]string{}
		state.readyState = xhrOpened
		vm.fireXHRReadyState(rt, obj, state)
		return goja.Undefined()
	})

	_ = obj.Set("setRequestHeader", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		value := call.Argument(1).String()
		state.headers[name] = value
		return goja.Undefined()
	})

	_ = obj.Set("send", func(call goja.FunctionCall) goja.Value {
		var body []byte
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
			body = []byte(call.Argument(0).String())
		}
		vm.xhrSend(rt, obj, state, body)
		return goja.Undefined()
	})

	_ = obj.Set("abort", func(call goja.FunctionCall) goja.Value {
		if state.cancel != nil {
			state.cancel()
		}
		if state.readyState != xhrUnsent && state.readyState != xhrDone {
			state.readyState = xhrDone
			state.status = 0
			vm.fireXHRReadyState(rt, obj, state)
			vm.fireXHREvent(rt, "abort", state.onAbort, state)
		}
		return goja.Undefined()
	})

	_ = obj.Set("getResponseHeader", func(call goja.FunctionCall) goja.Value {
		if state.respHeader == nil {
			return goja.Null()
		}
		name := call.Argument(0).String()
		values := state.respHeader.Values(name)
		if len(values) == 0 {
			return goja.Null()
		}
		return rt.ToValue(strings.Join(values, ", "))
	})

	_ = obj.Set("getAllResponseHeaders", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(formatAllResponseHeaders(state.respHeader))
	})

	_ = obj.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		cb, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			return goja.Undefined()
		}
		state.listeners[eventType] = append(state.listeners[eventType], cb)
		return goja.Undefined()
	})

	state.readyState = xhrUnsent
}

// formatAllResponseHeaders renders h in the XMLHttpRequest
// getAllResponseHeaders() format: "name: value\r\n" per header, lower-cased
// names, sorted for determinism (net/http.Header iteration order is
// randomized by Go's map iteration).
func formatAllResponseHeaders(h http.Header) string {
	if h == nil {
		return ""
	}
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		values := h.Values(name)
		fmt.Fprintf(&b, "%s: %s\r\n", strings.ToLower(name), strings.Join(values, ", "))
	}
	return b.String()
}
