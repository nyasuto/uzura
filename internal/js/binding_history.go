package js

import (
	"net/url"

	"github.com/dop251/goja"
)

// buildHistoryObject constructs the `history` object: pushState,
// replaceState, back, forward, go, plus the live `state`/`length` getters.
func buildHistoryObject(vm *VM, ls *locationState) *goja.Object {
	rt := vm.runtime
	obj := rt.NewObject()

	_ = obj.DefineAccessorProperty("length",
		rt.ToValue(func(call goja.FunctionCall) goja.Value {
			return rt.ToValue(len(ls.stack))
		}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("state",
		rt.ToValue(func(call goja.FunctionCall) goja.Value {
			return ls.stack[ls.idx].state
		}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.Set("pushState", func(call goja.FunctionCall) goja.Value {
		ls.pushState(call.Argument(0), call.Argument(2))
		return goja.Undefined()
	})

	_ = obj.Set("replaceState", func(call goja.FunctionCall) goja.Value {
		ls.replaceState(call.Argument(0), call.Argument(2))
		return goja.Undefined()
	})

	_ = obj.Set("back", func(call goja.FunctionCall) goja.Value {
		ls.navigateBy(-1)
		return goja.Undefined()
	})

	_ = obj.Set("forward", func(call goja.FunctionCall) goja.Value {
		ls.navigateBy(1)
		return goja.Undefined()
	})

	_ = obj.Set("go", func(call goja.FunctionCall) goja.Value {
		ls.navigateBy(int(call.Argument(0).ToInteger()))
		return goja.Undefined()
	})

	return obj
}

// resolve resolves ref against the current location URL, matching how a
// browser resolves the (often relative) url argument of pushState/
// replaceState / an <a href>.
func (ls *locationState) resolve(ref string) string {
	r, err := url.Parse(ref)
	if err != nil {
		return ls.u.String()
	}
	return ls.u.ResolveReference(r).String()
}

// pushState resolves urlArg (if given) against the current location,
// truncates any forward history entries, appends a new entry and makes it
// current. It updates location synchronously and does NOT fire popstate
// (matching the DOM spec: pushState/replaceState never dispatch popstate).
func (ls *locationState) pushState(state, urlArg goja.Value) {
	newURL := ls.u.String()
	if urlArg != nil && !goja.IsUndefined(urlArg) {
		newURL = ls.resolve(urlArg.String())
	}
	ls.applyURL(newURL)
	ls.stack = append(ls.stack[:ls.idx+1], historyEntry{state: state, rawURL: newURL})
	ls.idx = len(ls.stack) - 1
}

// replaceState behaves like pushState but overwrites the current entry
// in place instead of adding a new one.
func (ls *locationState) replaceState(state, urlArg goja.Value) {
	newURL := ls.u.String()
	if urlArg != nil && !goja.IsUndefined(urlArg) {
		newURL = ls.resolve(urlArg.String())
	}
	ls.applyURL(newURL)
	ls.stack[ls.idx] = historyEntry{state: state, rawURL: newURL}
}

// navigateBy moves the history cursor by delta entries (clamped to the
// stack bounds), updates location synchronously, and enqueues an async
// "popstate" dispatch carrying the target entry's state. A delta that
// would leave idx unchanged (already at a bound) is a no-op, matching
// real history.back()/forward() at the ends of the stack.
func (ls *locationState) navigateBy(delta int) {
	newIdx := ls.idx + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= len(ls.stack) {
		newIdx = len(ls.stack) - 1
	}
	if newIdx == ls.idx {
		return
	}
	ls.idx = newIdx
	entry := ls.stack[newIdx]
	ls.applyURL(entry.rawURL)

	state := entry.state
	ls.vm.loop.enqueueTask(func() {
		ls.firePopState(state)
	})
}

// applyURL parses raw and swaps it in as the current location URL,
// keeping vm's baseURL in sync so relative fetch()/XHR calls resolve as
// they would in a real SPA after a navigation.
func (ls *locationState) applyURL(raw string) {
	if u, err := url.Parse(raw); err == nil {
		ls.u = u
	}
	ls.vm.SetBaseURL(raw)
}

// firePopState dispatches a "popstate" event carrying state to window
// listeners and window.onpopstate.
func (ls *locationState) firePopState(state goja.Value) {
	rt := ls.vm.runtime
	evObj := rt.NewObject()
	_ = evObj.Set("type", "popstate")
	_ = evObj.Set("state", state)
	ls.dispatch("popstate", evObj)
}
