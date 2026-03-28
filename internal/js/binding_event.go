package js

import (
	"github.com/dop251/goja"
	"github.com/nyasuto/uzura/internal/dom"
)

type eventListener struct {
	callback goja.Callable
	cbValue  goja.Value
	capture  bool
}

type eventStore struct {
	listeners map[dom.Node]map[string][]eventListener
}

func newEventStore() *eventStore {
	return &eventStore{listeners: make(map[dom.Node]map[string][]eventListener)}
}

func (es *eventStore) add(node dom.Node, eventType string, cb goja.Callable, cbVal goja.Value, capture bool) {
	if es.listeners[node] == nil {
		es.listeners[node] = make(map[string][]eventListener)
	}
	es.listeners[node][eventType] = append(es.listeners[node][eventType], eventListener{
		callback: cb,
		cbValue:  cbVal,
		capture:  capture,
	})
}

func (es *eventStore) remove(node dom.Node, eventType string, cbVal goja.Value, capture bool) {
	listeners := es.listeners[node][eventType]
	for i, l := range listeners {
		if l.cbValue.SameAs(cbVal) && l.capture == capture {
			es.listeners[node][eventType] = append(listeners[:i], listeners[i+1:]...)
			return
		}
	}
}

func (es *eventStore) get(node dom.Node, eventType string) []eventListener {
	return es.listeners[node][eventType]
}

// addEventTargetMethods adds addEventListener, removeEventListener, and
// dispatchEvent to a JS object that wraps a dom.Node.
func (b *docBinder) addEventTargetMethods(obj *goja.Object, node dom.Node) {
	_ = obj.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		cb, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			return goja.Undefined()
		}
		capture := false
		if len(call.Arguments) > 2 {
			arg2 := call.Argument(2)
			if arg2.ExportType() != nil {
				capture = arg2.ToBoolean()
			}
		}
		b.events.add(node, eventType, cb, call.Argument(1), capture)
		return goja.Undefined()
	})

	_ = obj.Set("removeEventListener", func(call goja.FunctionCall) goja.Value {
		eventType := call.Argument(0).String()
		if _, ok := goja.AssertFunction(call.Argument(1)); !ok {
			return goja.Undefined()
		}
		capture := false
		if len(call.Arguments) > 2 {
			capture = call.Argument(2).ToBoolean()
		}
		b.events.remove(node, eventType, call.Argument(1), capture)
		return goja.Undefined()
	})

	_ = obj.Set("dispatchEvent", func(call goja.FunctionCall) goja.Value {
		evObj := call.Argument(0).ToObject(b.vm.runtime)
		b.dispatchEvent(node, evObj)
		return b.vm.runtime.ToValue(!evObj.Get("defaultPrevented").ToBoolean())
	})
}

func (b *docBinder) dispatchEvent(target dom.Node, evObj *goja.Object) {
	eventType := evObj.Get("type").String()
	bubbles := evObj.Get("bubbles").ToBoolean()
	_ = evObj.Set("target", b.wrapNode(target))

	stopped := false
	_ = evObj.Set("stopPropagation", func(call goja.FunctionCall) goja.Value {
		stopped = true
		return goja.Undefined()
	})

	// Build path from root to target
	var path []dom.Node
	for n := target; n != nil; n = n.ParentNode() {
		path = append(path, n)
	}

	// Capture phase (root → target, excluding target)
	for i := len(path) - 1; i > 0; i-- {
		if stopped {
			return
		}
		_ = evObj.Set("currentTarget", b.wrapNode(path[i]))
		for _, l := range b.events.get(path[i], eventType) {
			if l.capture {
				_, _ = l.callback(goja.Undefined(), b.vm.runtime.ToValue(evObj))
			}
		}
	}

	// Target phase
	if !stopped {
		_ = evObj.Set("currentTarget", b.wrapNode(target))
		for _, l := range b.events.get(target, eventType) {
			if stopped {
				return
			}
			_, _ = l.callback(goja.Undefined(), b.vm.runtime.ToValue(evObj))
		}
	}

	// Bubble phase (target parent → root)
	if bubbles {
		for i := 1; i < len(path); i++ {
			if stopped {
				return
			}
			_ = evObj.Set("currentTarget", b.wrapNode(path[i]))
			for _, l := range b.events.get(path[i], eventType) {
				if !l.capture {
					_, _ = l.callback(goja.Undefined(), b.vm.runtime.ToValue(evObj))
				}
			}
		}
	}
}

// setupEventConstructor registers the Event constructor on the VM global.
func (b *docBinder) setupEventConstructor() {
	_ = b.vm.runtime.Set("Event", func(call goja.ConstructorCall) *goja.Object {
		obj := call.This
		eventType := call.Argument(0).String()
		_ = obj.Set("type", eventType)
		_ = obj.Set("bubbles", false)
		_ = obj.Set("cancelable", false)
		_ = obj.Set("defaultPrevented", false)
		_ = obj.Set("target", goja.Null())
		_ = obj.Set("currentTarget", goja.Null())

		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Argument(1)) {
			opts := call.Argument(1).ToObject(b.vm.runtime)
			if v := opts.Get("bubbles"); v != nil && !goja.IsUndefined(v) {
				_ = obj.Set("bubbles", v.ToBoolean())
			}
			if v := opts.Get("cancelable"); v != nil && !goja.IsUndefined(v) {
				_ = obj.Set("cancelable", v.ToBoolean())
			}
		}

		_ = obj.Set("preventDefault", func(call goja.FunctionCall) goja.Value {
			if obj.Get("cancelable").ToBoolean() {
				_ = obj.Set("defaultPrevented", true)
			}
			return goja.Undefined()
		})
		_ = obj.Set("stopPropagation", func(call goja.FunctionCall) goja.Value {
			return goja.Undefined()
		})

		return obj
	})
}
