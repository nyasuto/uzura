package js

import (
	"github.com/dop251/goja"
	"github.com/nyasuto/uzura/internal/dom"
)

// BindDocument registers the document object and element proxies on the VM.
func BindDocument(vm *VM, doc *dom.Document) {
	b := &docBinder{vm: vm, doc: doc, events: newEventStore()}
	b.setupEventConstructor()
	_ = vm.runtime.Set("document", b.makeDocumentObj())
}

type docBinder struct {
	vm     *VM
	doc    *dom.Document
	events *eventStore
}

func (b *docBinder) makeDocumentObj() *goja.Object {
	obj := b.vm.runtime.NewObject()

	// nodeType = 9 (Document) — required for CDP subtype detection.
	_ = obj.Set("nodeType", 9)
	_ = obj.Set("nodeName", "#document")

	_ = obj.DefineAccessorProperty("title", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(b.doc.Title())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("documentElement", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		el := b.doc.DocumentElement()
		if el == nil {
			return goja.Null()
		}
		return b.wrapElement(el)
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("head", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		el := b.doc.Head()
		if el == nil {
			return goja.Null()
		}
		return b.wrapElement(el)
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("body", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		el := b.doc.Body()
		if el == nil {
			return goja.Null()
		}
		return b.wrapElement(el)
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.Set("getElementById", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		el := b.doc.GetElementById(id)
		if el == nil {
			return goja.Null()
		}
		return b.wrapElement(el)
	})

	_ = obj.Set("getElementsByTagName", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		elems := b.doc.GetElementsByTagName(name)
		return b.wrapElementList(elems)
	})

	_ = obj.Set("getElementsByClassName", func(call goja.FunctionCall) goja.Value {
		cls := call.Argument(0).String()
		elems := b.doc.GetElementsByClassName(cls)
		return b.wrapElementList(elems)
	})

	_ = obj.Set("querySelector", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		el, err := b.doc.QuerySelector(sel)
		if err != nil {
			panic(b.vm.runtime.NewGoError(err))
		}
		if el == nil {
			return goja.Null()
		}
		return b.wrapElement(el)
	})

	_ = obj.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		elems, err := b.doc.QuerySelectorAll(sel)
		if err != nil {
			panic(b.vm.runtime.NewGoError(err))
		}
		return b.wrapElementList(elems)
	})

	_ = obj.Set("createElement", func(call goja.FunctionCall) goja.Value {
		tag := call.Argument(0).String()
		el := b.doc.CreateElement(tag)
		return b.wrapElement(el)
	})

	_ = obj.Set("createTextNode", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()
		tn := b.doc.CreateTextNode(data)
		return b.wrapTextNode(tn)
	})

	_ = obj.Set("createDocumentFragment", func(call goja.FunctionCall) goja.Value {
		frag := b.doc.CreateDocumentFragment()
		return b.wrapFragment(frag)
	})

	b.addEventTargetMethods(obj, b.doc)

	return obj
}
