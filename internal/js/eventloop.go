package js

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/dop251/goja"
)

type eventLoop struct {
	mu      sync.Mutex
	timers  timerHeap
	nextID  int
	byID    map[int]*timerEntry
	tasks   []func()
	pending int
	wake    chan struct{} // buffered(1): signals a task was added or pending decreased

	// sawActivity is set once any one-shot timer, task, or pending async
	// operation has ever been registered on this loop. setInterval alone
	// never sets it. It gates whether a still-live repeating interval is
	// allowed to be ignored when deciding to settle (see run): a VM used
	// standalone with nothing but a bare setInterval must keep behaving
	// like a real timer (waiting for/firing it) rather than exiting before
	// it ever fires, but once the loop has done real one-shot/async work
	// and that work has drained, a live interval alone must not hold it
	// hostage — see TestEventLoop_SettlesWithLiveInterval.
	sawActivity bool

	// ctx is the context of the currently running loop iteration (set by
	// RunEventLoopContext via setContext, read by LoopContext from any goroutine,
	// including fetch/XHR worker goroutines). Guarded by mu so there is a proper
	// happens-before edge between the loop clearing it and other goroutines
	// observing the clear — a bare field here previously raced under -race.
	ctx context.Context
}

func newEventLoop() *eventLoop {
	return &eventLoop{
		byID: make(map[int]*timerEntry),
		wake: make(chan struct{}, 1),
	}
}

func (el *eventLoop) signalWake() {
	select {
	case el.wake <- struct{}{}:
	default:
	}
}

// enqueueTask schedules fn to run on the loop thread.
func (el *eventLoop) enqueueTask(fn func()) {
	el.mu.Lock()
	el.tasks = append(el.tasks, fn)
	el.sawActivity = true
	el.mu.Unlock()
	el.signalWake()
}

// addPending marks one in-flight async operation (e.g. an HTTP request) as
// started. Every addPending call must be balanced by exactly one matching
// donePending (or completeWith) call; an over-decrement of pending below
// zero makes the loop treat itself as perpetually "still busy" and hang
// until ctx cancellation, since it will never observe pending == 0.
//
// Ordering hazard: if the async operation's result must be delivered to the
// loop via enqueueTask, do NOT call enqueueTask and donePending separately
// from a background goroutine — the loop can observe pending == 0 with an
// empty queue between the two calls and settle, orphaning the enqueued
// task. Use completeWith instead, which performs both under one lock.
func (el *eventLoop) addPending() {
	el.mu.Lock()
	el.pending++
	el.sawActivity = true
	el.mu.Unlock()
}

// donePending marks one in-flight async operation as finished. See addPending
// for the balancing requirement (pending must never go negative) and the
// completeWith ordering hazard when a task also needs to be enqueued.
func (el *eventLoop) donePending() {
	el.mu.Lock()
	el.pending--
	el.mu.Unlock()
	el.signalWake()
}

func (el *eventLoop) setTimeout(cb goja.Callable, delay time.Duration) int {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.nextID++
	entry := &timerEntry{
		id:       el.nextID,
		callback: cb,
		fireAt:   time.Now().Add(delay),
	}
	heap.Push(&el.timers, entry)
	el.byID[entry.id] = entry
	el.sawActivity = true
	return entry.id
}

func (el *eventLoop) setInterval(cb goja.Callable, interval time.Duration) int {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.nextID++
	entry := &timerEntry{
		id:       el.nextID,
		callback: cb,
		fireAt:   time.Now().Add(interval),
		interval: interval,
	}
	heap.Push(&el.timers, entry)
	el.byID[entry.id] = entry
	return entry.id
}

func (el *eventLoop) clearTimer(id int) {
	el.mu.Lock()
	defer el.mu.Unlock()
	if entry, ok := el.byID[id]; ok {
		entry.cleared = true
		delete(el.byID, id)
	}
}

// run processes tasks and timers until the loop is quiescent or ctx is
// done. Quiescent means: no queued tasks, no pending async operation
// (fetch/XHR), and no *one-shot* timer left to fire. A repeating
// setInterval that is never cleared (e.g. an analytics heartbeat or a CSS
// animation driver) does NOT by itself keep the loop alive — once the rest
// of a page's work has settled, run returns even though the interval is
// still registered. Firing that interval forever "waiting" for it to go
// away would defeat the point: it never will on its own, so the loop would
// otherwise only ever exit via ctx cancellation, hanging navigation for the
// full deadline on ordinary, harmless pages.
func (el *eventLoop) run(ctx context.Context) error {
	for {
		// 1. Drain the task queue first (tasks beat not-yet-due timers).
		for {
			el.mu.Lock()
			if len(el.tasks) == 0 {
				el.mu.Unlock()
				break
			}
			task := el.tasks[0]
			el.tasks = el.tasks[1:]
			el.mu.Unlock()
			task()
		}

		// 2. Fire due timers (one at a time so new tasks get priority).
		el.mu.Lock()
		var wait time.Duration = -1
		if el.timers.Len() > 0 {
			entry := el.timers[0]
			if entry.cleared {
				heap.Pop(&el.timers)
				el.mu.Unlock()
				continue
			}
			now := time.Now()
			if !entry.fireAt.After(now) {
				heap.Pop(&el.timers)
				cb := entry.callback
				isInterval := entry.interval > 0
				el.mu.Unlock()
				_, _ = cb(goja.Undefined())
				if isInterval {
					el.mu.Lock()
					if !entry.cleared {
						entry.fireAt = time.Now().Add(entry.interval)
						heap.Push(&el.timers, entry)
					}
					el.mu.Unlock()
				}
				continue
			}
			wait = entry.fireAt.Sub(now)
		}
		hasWork := len(el.tasks) > 0
		pending := el.pending
		hasOneShotTimer := false
		hasLiveInterval := false
		for _, entry := range el.timers {
			if entry.cleared {
				continue
			}
			if entry.interval > 0 {
				hasLiveInterval = true
			} else {
				hasOneShotTimer = true
			}
		}
		sawActivity := el.sawActivity
		el.mu.Unlock()

		if hasWork {
			continue
		}
		// Settled once nothing beats a still-registered interval: no queued
		// tasks, no pending async op, no one-shot timer left to fire, and
		// either no interval is live or (once real one-shot/async activity
		// has happened and drained) we stop letting it hold the loop
		// hostage. sawActivity keeps a VM used standalone with nothing but
		// a bare setInterval behaving like a normal timer instead of never
		// firing it at all.
		if pending == 0 && !hasOneShotTimer && (!hasLiveInterval || sawActivity) {
			return nil
		}

		// 3. Wait for: next timer, a wake signal, or cancellation.
		var timerC <-chan time.Time
		var timer *time.Timer
		if wait >= 0 {
			timer = time.NewTimer(wait)
			timerC = timer.C
		}
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return ctx.Err()
		case <-el.wake:
		case <-timerC:
		}
		if timer != nil {
			timer.Stop()
		}
	}
}

func (vm *VM) setupTimers() {
	el := newEventLoop()
	vm.loop = el

	_ = vm.runtime.Set("setTimeout", func(call goja.FunctionCall) goja.Value {
		cb, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.runtime.NewTypeError("setTimeout: callback is not a function"))
		}
		delay := time.Duration(call.Argument(1).ToInteger()) * time.Millisecond
		id := el.setTimeout(cb, delay)
		return vm.runtime.ToValue(id)
	})

	_ = vm.runtime.Set("clearTimeout", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		el.clearTimer(id)
		return goja.Undefined()
	})

	_ = vm.runtime.Set("setInterval", func(call goja.FunctionCall) goja.Value {
		cb, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(vm.runtime.NewTypeError("setInterval: callback is not a function"))
		}
		interval := time.Duration(call.Argument(1).ToInteger()) * time.Millisecond
		id := el.setInterval(cb, interval)
		return vm.runtime.ToValue(id)
	})

	_ = vm.runtime.Set("clearInterval", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		el.clearTimer(id)
		return goja.Undefined()
	})
}
