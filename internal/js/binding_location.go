package js

import (
	"net/url"
	"strings"

	"github.com/dop251/goja"
)

// historyEntry is one entry in the session history stack.
type historyEntry struct {
	state  goja.Value
	rawURL string
}

// winListener pairs a callable with its original JS value so it can be
// matched (and removed) by identity via SameAs.
type winListener struct {
	cb  goja.Callable
	val goja.Value
}

// locationState holds the shared state behind location, history and the
// window-level popstate/hashchange event registrations. u is the live
// "current URL"; all location getters recompute from it on every access.
type locationState struct {
	vm        *VM
	u         *url.URL
	stack     []historyEntry
	idx       int
	listeners map[string][]winListener
}

// BindLocation registers `location` (aliased onto document.location too),
// `history`, and window-level addEventListener/removeEventListener for
// "popstate"/"hashchange" on vm. It also calls vm.SetBaseURL(rawURL) so
// fetch/XHR resolve relative URLs against the current document location,
// and keeps baseURL following the location on every subsequent navigation
// (pushState/replaceState/back/forward/go/hash assignment) so relative
// fetches keep resolving correctly, like in a real SPA.
func BindLocation(vm *VM, rawURL string) {
	vm.SetBaseURL(rawURL)

	u, err := url.Parse(rawURL)
	if err != nil {
		u = &url.URL{}
	}

	ls := &locationState{
		vm:        vm,
		u:         u,
		stack:     []historyEntry{{state: goja.Null(), rawURL: rawURL}},
		idx:       0,
		listeners: make(map[string][]winListener),
	}

	rt := vm.runtime

	locObj := buildLocationObject(vm, ls)
	_ = rt.Set("location", locObj)
	if docVal := rt.Get("document"); docVal != nil {
		if docObj, ok := docVal.(*goja.Object); ok {
			_ = docObj.Set("location", locObj)
		}
	}

	_ = rt.Set("history", buildHistoryObject(vm, ls))

	setupWindowEvents(vm, ls)
}

// buildLocationObject constructs the `location` object. All parts are
// live getters (DefineAccessorProperty) recomputed from ls.u; only `hash`
// has a setter, which updates ls.u and dispatches "hashchange" async.
func buildLocationObject(vm *VM, ls *locationState) *goja.Object {
	rt := vm.runtime
	obj := rt.NewObject()

	getter := func(fn func() string) goja.Value {
		return rt.ToValue(func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(fn())
		})
	}

	define := func(name string, fn func() string) {
		_ = obj.DefineAccessorProperty(name, getter(fn), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)
	}

	define("href", func() string { return ls.u.String() })
	define("origin", func() string { return ls.u.Scheme + "://" + ls.u.Host })
	define("protocol", func() string { return ls.u.Scheme + ":" })
	define("host", func() string { return ls.u.Host })
	define("hostname", func() string { return ls.u.Hostname() })
	define("port", func() string { return ls.u.Port() })
	define("pathname", func() string { return ls.u.Path })
	define("search", func() string {
		if ls.u.RawQuery == "" {
			return ""
		}
		return "?" + ls.u.RawQuery
	})

	hashGetter := getter(func() string {
		if ls.u.Fragment == "" {
			return ""
		}
		return "#" + ls.u.Fragment
	})
	hashSetter := rt.ToValue(func(call goja.FunctionCall) goja.Value {
		newFrag := strings.TrimPrefix(call.Argument(0).String(), "#")
		if newFrag == ls.u.Fragment {
			return goja.Undefined()
		}
		ls.u.Fragment = newFrag
		ls.u.RawFragment = ""
		vm.SetBaseURL(ls.u.String())
		ls.fireHashChange()
		return goja.Undefined()
	})
	_ = obj.DefineAccessorProperty("hash", hashGetter, hashSetter, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.Set("toString", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(ls.u.String())
	})

	for _, name := range []string{"assign", "replace", "reload"} {
		method := name
		_ = obj.Set(method, func(call goja.FunctionCall) goja.Value {
			warnUnsupported(vm, "location."+method+"() is not supported")
			return goja.Undefined()
		})
	}

	return obj
}

// warnUnsupported routes a message through the console.warn binding, so it
// is visible in the same channel as any other JS console output.
func warnUnsupported(vm *VM, msg string) {
	rt := vm.runtime
	consoleVal, ok := rt.Get("console").(*goja.Object)
	if !ok {
		return
	}
	warnFn, ok := goja.AssertFunction(consoleVal.Get("warn"))
	if !ok {
		return
	}
	_, _ = warnFn(goja.Undefined(), rt.ToValue(msg))
}

// setupWindowEvents registers global addEventListener/removeEventListener
// functions (window === globalThis, so a global function is a window
// method). Only "popstate" and "hashchange" are recognized; other event
// types are silently ignored, independent of document's own EventTarget
// implementation.
func setupWindowEvents(vm *VM, ls *locationState) {
	rt := vm.runtime

	_ = rt.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		if eventType != "popstate" && eventType != "hashchange" {
			return goja.Undefined()
		}
		cb, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			return goja.Undefined()
		}
		ls.listeners[eventType] = append(ls.listeners[eventType], winListener{cb: cb, val: call.Argument(1)})
		return goja.Undefined()
	})

	_ = rt.Set("removeEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		list := ls.listeners[eventType]
		for i, l := range list {
			if l.val.SameAs(call.Argument(1)) {
				ls.listeners[eventType] = append(list[:i], list[i+1:]...)
				break
			}
		}
		return goja.Undefined()
	})
}

// dispatch invokes every registered listener for eventType, then the
// corresponding on<eventType> global property if it is callable.
func (ls *locationState) dispatch(eventType string, evObj *goja.Object) {
	rt := ls.vm.runtime
	for _, l := range ls.listeners[eventType] {
		_, _ = l.cb(goja.Undefined(), evObj)
	}
	if fn, ok := goja.AssertFunction(rt.Get("on" + eventType)); ok {
		_, _ = fn(goja.Undefined(), evObj)
	}
}

// fireHashChange enqueues an async "hashchange" dispatch on the event loop.
func (ls *locationState) fireHashChange() {
	rt := ls.vm.runtime
	evObj := rt.NewObject()
	_ = evObj.Set("type", "hashchange")
	ls.vm.loop.enqueueTask(func() {
		ls.dispatch("hashchange", evObj)
	})
}
