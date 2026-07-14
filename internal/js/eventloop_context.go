package js

import "context"

// Event loop context plumbing and completion helpers.

// setContext records ctx as the context of the currently running loop
// iteration. Guarded by mu; safe to call concurrently with loopContext.
func (el *eventLoop) setContext(ctx context.Context) {
	el.mu.Lock()
	el.ctx = ctx
	el.mu.Unlock()
}

// clearContext resets the stored loop context once the loop has stopped
// running, so loopContext falls back to context.Background().
func (el *eventLoop) clearContext() {
	el.mu.Lock()
	el.ctx = nil
	el.mu.Unlock()
}

// loopContext returns the context of the currently running loop, or
// context.Background() if no loop is running. Safe to call from any
// goroutine, including fetch/XHR worker goroutines started while the loop
// is running.
func (el *eventLoop) loopContext() context.Context {
	el.mu.Lock()
	ctx := el.ctx
	el.mu.Unlock()
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// completeWith enqueues fn and marks one pending async operation as
// finished in a single critical section, so the loop can never settle
// between the two. Prefer this over separate enqueueTask+donePending calls
// whenever an async operation (e.g. fetch/XHR) delivers its result via a
// task: performing both under one lock acquisition closes the window where
// the loop could observe pending == 0 with an empty task queue and exit,
// orphaning fn.
func (el *eventLoop) completeWith(fn func()) {
	el.mu.Lock()
	el.tasks = append(el.tasks, fn)
	el.pending--
	el.mu.Unlock()
	el.signalWake()
}

// RunEventLoopContext processes timers, tasks and in-flight async work
// until everything settles or ctx is done. Returns ctx.Err() on cancellation.
//
// Not reentrant: RunEventLoopContext must not be called again (e.g. from a
// nested/recursive call while one is already running on this VM's loop) —
// a nested call clobbers the stored context that LoopContext hands out to
// concurrently running fetch/XHR goroutines.
func (vm *VM) RunEventLoopContext(ctx context.Context) error {
	if vm.loop == nil {
		return nil
	}
	vm.loop.setContext(ctx)
	defer vm.loop.clearContext()
	return vm.loop.run(ctx)
}

// RunEventLoop processes all pending timers and callbacks until the queue is empty.
func (vm *VM) RunEventLoop() {
	_ = vm.RunEventLoopContext(context.Background())
}

// LoopContext returns the context of the currently running event loop, or
// context.Background() if no loop is running. Safe to call concurrently
// from any goroutine — async bindings (fetch/XHR) use it for their HTTP
// requests from worker goroutines while the loop runs on another one.
func (vm *VM) LoopContext() context.Context {
	if vm.loop == nil {
		return context.Background()
	}
	return vm.loop.loopContext()
}
