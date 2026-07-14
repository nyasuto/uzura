package js

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// setXHRAccessors installs the readyState/status/statusText/response*/
// timeout/on* properties on obj as live accessors backed by state, so JS
// always observes the current value rather than a construction-time
// snapshot.
func (vm *VM) setXHRAccessors(obj *goja.Object, state *xhrState) {
	rt := vm.runtime

	_ = obj.DefineAccessorProperty("readyState",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return rt.ToValue(state.readyState) }),
		nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("status",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return rt.ToValue(state.status) }),
		nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("statusText",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return rt.ToValue(state.statusText) }),
		nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("responseText",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return rt.ToValue(string(state.body)) }),
		nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("response",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return vm.xhrResponseValue(state) }),
		nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("responseType",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return rt.ToValue(state.responseType) }),
		rt.ToValue(func(call goja.FunctionCall) goja.Value {
			state.responseType = call.Argument(0).String()
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("timeout",
		rt.ToValue(func(goja.FunctionCall) goja.Value { return rt.ToValue(state.timeout.Milliseconds()) }),
		rt.ToValue(func(call goja.FunctionCall) goja.Value {
			state.timeout = time.Duration(call.Argument(0).ToInteger()) * time.Millisecond
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	setXHRCallbackAccessor(obj, rt, "onreadystatechange", &state.onReadyStateChange)
	setXHRCallbackAccessor(obj, rt, "onload", &state.onLoad)
	setXHRCallbackAccessor(obj, rt, "onerror", &state.onError)
	setXHRCallbackAccessor(obj, rt, "ontimeout", &state.onTimeout)
	setXHRCallbackAccessor(obj, rt, "onabort", &state.onAbort)
}

// setXHRCallbackAccessor wires an on* property so assigning a JS function
// stores it (as a goja.Callable) in *slot, and reading it back returns
// goja.Null() when unset.
func setXHRCallbackAccessor(obj *goja.Object, rt *goja.Runtime, name string, slot *goja.Callable) {
	_ = obj.DefineAccessorProperty(name,
		rt.ToValue(func(goja.FunctionCall) goja.Value {
			if *slot == nil {
				return goja.Null()
			}
			return rt.ToValue(*slot)
		}),
		rt.ToValue(func(call goja.FunctionCall) goja.Value {
			if cb, ok := goja.AssertFunction(call.Argument(0)); ok {
				*slot = cb
			} else {
				*slot = nil
			}
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE,
	)
}

// xhrResponseValue computes the `response` property per responseType:
// ”/'text' return the raw body string; 'json' parses it, yielding null on
// a parse failure (per spec, `response` is null rather than throwing).
func (vm *VM) xhrResponseValue(state *xhrState) goja.Value {
	rt := vm.runtime
	switch state.responseType {
	case "", "text":
		return rt.ToValue(string(state.body))
	case "json":
		var data interface{}
		if err := json.Unmarshal(state.body, &data); err != nil {
			return goja.Null()
		}
		return rt.ToValue(data)
	default:
		return rt.ToValue(string(state.body))
	}
}

// fireXHRReadyState invokes onreadystatechange plus any
// "readystatechange" addEventListener callbacks, in that order.
func (vm *VM) fireXHRReadyState(rt *goja.Runtime, obj *goja.Object, state *xhrState) {
	self := rt.ToValue(obj)
	if state.onReadyStateChange != nil {
		_, _ = state.onReadyStateChange(self)
	}
	for _, cb := range state.listeners["readystatechange"] {
		_, _ = cb(self)
	}
}

// fireXHREvent invokes the on* callback (if set) followed by any
// addEventListener callbacks registered for eventType.
func (vm *VM) fireXHREvent(rt *goja.Runtime, eventType string, onCb goja.Callable, state *xhrState) {
	if onCb != nil {
		_, _ = onCb(goja.Undefined())
	}
	for _, cb := range state.listeners[eventType] {
		_, _ = cb(goja.Undefined())
	}
}

// xhrSend performs the network request for state, transitioning readyState
// through HEADERS_RECEIVED(2) -> LOADING(3) -> DONE(4) in a single
// completeWith task once the response (or error) arrives, per the brief:
// real browsers may deliver these across separate task-queue turns, uzura
// does not.
func (vm *VM) xhrSend(rt *goja.Runtime, obj *goja.Object, state *xhrState, body []byte) {
	req := HTTPRequest{
		Method:  state.method,
		URL:     state.url,
		Headers: state.headers,
		Body:    body,
	}

	client := vm.httpClient()
	ctx, cancel := context.WithCancel(vm.LoopContext())
	state.cancel = cancel

	var timeoutCtx context.Context
	var timeoutCancel context.CancelFunc
	reqCtx := ctx
	if state.timeout > 0 {
		timeoutCtx, timeoutCancel = context.WithTimeout(ctx, state.timeout)
		reqCtx = timeoutCtx
	}
	state.timeoutCtx = timeoutCtx

	vm.loop.addPending()
	go func() {
		var resp *HTTPResponse
		var err error
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					err = fmt.Errorf("xhr: %v", rec)
				}
			}()
			resp, err = client(reqCtx, req)
		}()
		vm.loop.completeWith(func() {
			cancel()
			if timeoutCancel != nil {
				timeoutCancel()
			}
			vm.deliverXHRResult(rt, obj, state, resp, err)
		})
	}()
}

// deliverXHRResult runs on the loop thread once xhrSend's goroutine
// completes. It applies the HEADERS_RECEIVED/LOADING/DONE transitions and
// fires the terminal event (load/error/timeout).
func (vm *VM) deliverXHRResult(rt *goja.Runtime, obj *goja.Object, state *xhrState, resp *HTTPResponse, err error) {
	// abort() already drove readyState to DONE and fired onabort
	// synchronously; skip firing a second terminal event when this
	// goroutine's result arrives afterwards.
	if state.readyState == xhrDone {
		return
	}
	if err != nil {
		state.readyState = xhrDone
		state.status = 0
		vm.fireXHRReadyState(rt, obj, state)
		if isTimeoutErr(err, state) {
			vm.fireXHREvent(rt, "timeout", state.onTimeout, state)
		} else {
			vm.fireXHREvent(rt, "error", state.onError, state)
		}
		return
	}

	state.status = resp.Status
	state.statusText = resp.StatusText
	state.respHeader = resp.Headers
	state.body = resp.Body

	state.readyState = xhrHeadersReceived
	vm.fireXHRReadyState(rt, obj, state)

	state.readyState = xhrLoading
	vm.fireXHRReadyState(rt, obj, state)

	state.readyState = xhrDone
	vm.fireXHRReadyState(rt, obj, state)

	vm.fireXHREvent(rt, "load", state.onLoad, state)
}

// isTimeoutErr reports whether err originates from this XHR's own
// state.timeout deadline (state.timeoutCtx), as opposed to a generic
// transport failure or an outer/page-level deadline threaded in via
// RunEventLoopContext — both of which also surface as
// context.DeadlineExceeded but must be reported via onerror instead. Only
// state.timeoutCtx's own expiry counts as this XHR's timeout; a bare
// context.DeadlineExceeded is not sufficient evidence on its own since it
// also propagates from the page-level context when no per-request timeout
// was set.
func isTimeoutErr(err error, state *xhrState) bool {
	if state.timeout <= 0 || state.timeoutCtx == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) && errors.Is(state.timeoutCtx.Err(), context.DeadlineExceeded)
}
