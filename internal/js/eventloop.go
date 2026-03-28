package js

import (
	"container/heap"
	"sync"
	"time"

	"github.com/dop251/goja"
)

type timerEntry struct {
	id       int
	callback goja.Callable
	fireAt   time.Time
	interval time.Duration
	cleared  bool
	index    int
}

type timerHeap []*timerEntry

func (h timerHeap) Len() int           { return len(h) }
func (h timerHeap) Less(i, j int) bool { return h[i].fireAt.Before(h[j].fireAt) }

func (h timerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *timerHeap) Push(x interface{}) {
	e, _ := x.(*timerEntry)
	e.index = len(*h)
	*h = append(*h, e)
}

func (h *timerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	e.index = -1
	*h = old[:n-1]
	return e
}

type eventLoop struct {
	mu     sync.Mutex
	timers timerHeap
	nextID int
	byID   map[int]*timerEntry
}

func newEventLoop() *eventLoop {
	return &eventLoop{
		byID: make(map[int]*timerEntry),
	}
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

// run processes all pending timers until none remain.
func (el *eventLoop) run(runtime *goja.Runtime) {
	for {
		el.mu.Lock()
		if el.timers.Len() == 0 {
			el.mu.Unlock()
			return
		}
		entry := el.timers[0]
		if entry.cleared {
			heap.Pop(&el.timers)
			el.mu.Unlock()
			continue
		}

		now := time.Now()
		if entry.fireAt.After(now) {
			wait := entry.fireAt.Sub(now)
			el.mu.Unlock()
			time.Sleep(wait)
			continue
		}

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

		// Check if runtime context is still valid
		_ = runtime
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

// RunEventLoop processes all pending timers and callbacks until the queue is empty.
func (vm *VM) RunEventLoop() {
	if vm.loop != nil {
		vm.loop.run(vm.runtime)
	}
}
