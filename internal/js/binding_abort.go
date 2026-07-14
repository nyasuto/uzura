package js

import (
	"context"

	"github.com/dop251/goja"
)

// abortState is the Go-side state backing one AbortController/AbortSignal
// pair. It is only ever mutated from the event-loop thread: abort() is a
// synchronous JS call, so there is no concurrent access to guard against
// here. Goroutines (e.g. fetch's worker goroutine) never touch abortState
// directly — they observe cancellation only through the context.Context
// whose CancelFunc is registered in cancels.
type abortState struct {
	aborted   bool
	cancels   []context.CancelFunc
	listeners []goja.Callable
}

// registerCancel adds cancel to the set of functions invoked when this
// signal aborts. If the signal is already aborted, cancel runs immediately.
func (s *abortState) registerCancel(cancel context.CancelFunc) {
	if s.aborted {
		cancel()
		return
	}
	s.cancels = append(s.cancels, cancel)
}

// abort marks the state as aborted, calls every registered cancel func, and
// invokes every "abort" listener. Safe to call more than once; only the
// first call has any effect.
func (s *abortState) abort() {
	if s.aborted {
		return
	}
	s.aborted = true
	for _, cancel := range s.cancels {
		cancel()
	}
	for _, l := range s.listeners {
		_, _ = l(goja.Undefined())
	}
}

// BindAbort registers the AbortController and AbortSignal constructors on
// the VM's global object. Signal instances carry their Go-side abortState
// in vm.abortStates, keyed by the signal's *goja.Object identity — this
// keeps the state out of JS entirely (no hidden property to accidentally
// enumerate or overwrite) at the cost of the map living for the VM's
// lifetime; that's an acceptable trade-off since a VM handles one page's
// script lifetime, not a long-running process.
func BindAbort(vm *VM) {
	rt := vm.runtime

	_ = rt.Set("AbortController", func(call goja.ConstructorCall) *goja.Object {
		state := &abortState{}
		signal := newAbortSignalObject(vm, state)
		vm.setAbortState(signal, state)

		ctrl := call.This
		_ = ctrl.Set("signal", signal)
		_ = ctrl.Set("abort", func(goja.FunctionCall) goja.Value {
			state.abort()
			return goja.Undefined()
		})
		return ctrl
	})
}

// setAbortState associates state with signal for later lookup by fetch's
// init.signal handling. See BindAbort for why a VM-level map is used
// instead of a hidden JS property.
func (vm *VM) setAbortState(signal *goja.Object, state *abortState) {
	if vm.abortStates == nil {
		vm.abortStates = make(map[*goja.Object]*abortState)
	}
	vm.abortStates[signal] = state
}

// abortStateFor returns the abortState associated with a signal value, or
// nil if v is not a known AbortSignal object (e.g. BindAbort was never
// called, or the value is something else entirely).
func (vm *VM) abortStateFor(v goja.Value) *abortState {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	return vm.abortStates[v.ToObject(vm.runtime)]
}

// newAbortSignalObject builds a plain JS object exposing the AbortSignal
// surface (`aborted`, `addEventListener("abort", cb)`) backed by state.
// `aborted` is a live accessor property so it always reflects the current
// value of state.aborted rather than a snapshot taken at construction time.
func newAbortSignalObject(vm *VM, state *abortState) *goja.Object {
	rt := vm.runtime
	signal := rt.NewObject()

	_ = signal.DefineAccessorProperty("aborted",
		rt.ToValue(func(goja.FunctionCall) goja.Value {
			return rt.ToValue(state.aborted)
		}),
		nil,
		goja.FLAG_FALSE, goja.FLAG_TRUE,
	)

	_ = signal.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		if eventType != "abort" {
			return goja.Undefined()
		}
		cb, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			return goja.Undefined()
		}
		state.listeners = append(state.listeners, cb)
		return goja.Undefined()
	})

	return signal
}
