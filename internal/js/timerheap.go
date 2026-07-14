package js

import (
	"time"

	"github.com/dop251/goja"
)

// timerEntry is one scheduled setTimeout/setInterval callback.
type timerEntry struct {
	id       int
	callback goja.Callable
	fireAt   time.Time
	interval time.Duration
	cleared  bool
	index    int
}

// timerHeap is a container/heap.Interface min-heap of timerEntry ordered by
// fireAt, so the next timer to fire is always at index 0.
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
