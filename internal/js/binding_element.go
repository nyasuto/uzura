package js

import (
	"strconv"

	"github.com/dop251/goja"
	"github.com/nyasuto/uzura/internal/dom"
)

func (b *docBinder) wrapElement(el *dom.Element) goja.Value {
	obj := b.vm.runtime.NewObject()

	// nodeType = 1 (Element) — required for CDP subtype detection.
	_ = obj.Set("nodeType", 1)
	_ = obj.Set("nodeName", el.TagName())

	_ = obj.DefineAccessorProperty("tagName", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.TagName())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("id", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.Id())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("className", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.ClassName())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("textContent",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			return b.vm.runtime.ToValue(el.TextContent())
		}),
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			el.SetTextContent(call.Argument(0).String())
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("innerHTML",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			return b.vm.runtime.ToValue(dom.InnerHTML(el))
		}),
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			if err := el.SetInnerHTML(call.Argument(0).String()); err != nil {
				panic(b.vm.runtime.NewGoError(err))
			}
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("outerHTML", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(dom.OuterHTML(el))
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.Set("getAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		if !el.HasAttribute(name) {
			return goja.Null()
		}
		return b.vm.runtime.ToValue(el.GetAttribute(name))
	})

	_ = obj.Set("hasAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		return b.vm.runtime.ToValue(el.HasAttribute(name))
	})

	_ = obj.Set("querySelector", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		found, err := el.QuerySelector(sel)
		if err != nil {
			panic(b.vm.runtime.NewGoError(err))
		}
		if found == nil {
			return goja.Null()
		}
		return b.wrapElement(found)
	})

	_ = obj.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		elems, err := el.QuerySelectorAll(sel)
		if err != nil {
			panic(b.vm.runtime.NewGoError(err))
		}
		return b.wrapElementList(elems)
	})

	_ = obj.Set("matches", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		ok, err := el.Matches(sel)
		if err != nil {
			panic(b.vm.runtime.NewGoError(err))
		}
		return b.vm.runtime.ToValue(ok)
	})

	_ = obj.Set("closest", func(call goja.FunctionCall) goja.Value {
		sel := call.Argument(0).String()
		found, err := el.Closest(sel)
		if err != nil {
			panic(b.vm.runtime.NewGoError(err))
		}
		if found == nil {
			return goja.Null()
		}
		return b.wrapElement(found)
	})

	_ = obj.Set("setAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		value := call.Argument(1).String()
		el.SetAttribute(name, value)
		return goja.Undefined()
	})

	_ = obj.Set("removeAttribute", func(call goja.FunctionCall) goja.Value {
		name := call.Argument(0).String()
		el.RemoveAttribute(name)
		return goja.Undefined()
	})

	// value property for form elements (input, textarea, select).
	// Backed by the DOM "value" attribute for persistence across wrapElement calls.
	_ = obj.DefineAccessorProperty("value",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			return b.vm.runtime.ToValue(el.GetAttribute("value"))
		}),
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			el.SetAttribute("value", call.Argument(0).String())
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	// checked property for checkbox/radio inputs.
	_ = obj.DefineAccessorProperty("checked",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			return b.vm.runtime.ToValue(el.HasAttribute("checked"))
		}),
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			if call.Argument(0).ToBoolean() {
				el.SetAttribute("checked", "")
			} else {
				el.RemoveAttribute("checked")
			}
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)

	b.addNodeMethods(obj, el)
	b.addClassListProperty(obj, el)
	b.addDatasetProperty(obj, el)
	b.addEventTargetMethods(obj, el)
	_ = obj.Set("_goNode", el)

	return b.vm.runtime.ToValue(obj)
}

func (b *docBinder) wrapElementList(elems []*dom.Element) goja.Value {
	obj := b.vm.runtime.NewObject()

	_ = obj.DefineAccessorProperty("length", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(len(elems))
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	for i, el := range elems {
		_ = obj.Set(intToStr(i), b.wrapElement(el))
	}

	_ = obj.Set("forEach", func(call goja.FunctionCall) goja.Value {
		cb, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			panic(b.vm.runtime.NewTypeError("forEach callback is not a function"))
		}
		for i, el := range elems {
			_, err := cb(goja.Undefined(), b.wrapElement(el), b.vm.runtime.ToValue(i), b.vm.runtime.ToValue(obj))
			if err != nil {
				panic(err)
			}
		}
		return goja.Undefined()
	})

	_ = obj.Set("item", func(call goja.FunctionCall) goja.Value {
		idx := int(call.Argument(0).ToInteger())
		if idx < 0 || idx >= len(elems) {
			return goja.Null()
		}
		return b.wrapElement(elems[idx])
	})

	return b.vm.runtime.ToValue(obj)
}

func intToStr(i int) string {
	return strconv.Itoa(i)
}

func (b *docBinder) addNodeMethods(obj *goja.Object, node dom.Node) {
	_ = obj.DefineAccessorProperty("parentNode",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			p := node.ParentNode()
			if p == nil {
				return goja.Null()
			}
			return b.wrapNode(p)
		}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.Set("appendChild", func(call goja.FunctionCall) goja.Value {
		child := b.unwrapNode(call.Argument(0))
		node.AppendChild(child)
		return call.Argument(0)
	})

	_ = obj.Set("removeChild", func(call goja.FunctionCall) goja.Value {
		child := b.unwrapNode(call.Argument(0))
		node.RemoveChild(child)
		return call.Argument(0)
	})

	_ = obj.Set("insertBefore", func(call goja.FunctionCall) goja.Value {
		newChild := b.unwrapNode(call.Argument(0))
		var refChild dom.Node
		if !goja.IsNull(call.Argument(1)) && !goja.IsUndefined(call.Argument(1)) {
			refChild = b.unwrapNode(call.Argument(1))
		}
		node.InsertBefore(newChild, refChild)
		return call.Argument(0)
	})

	_ = obj.Set("replaceChild", func(call goja.FunctionCall) goja.Value {
		newChild := b.unwrapNode(call.Argument(0))
		oldChild := b.unwrapNode(call.Argument(1))
		node.ReplaceChild(newChild, oldChild)
		return call.Argument(1)
	})
}

func (b *docBinder) addClassListProperty(obj *goja.Object, el *dom.Element) {
	_ = obj.DefineAccessorProperty("classList",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			cl := el.ClassList()
			clObj := b.vm.runtime.NewObject()

			_ = clObj.Set("add", func(c goja.FunctionCall) goja.Value {
				for _, arg := range c.Arguments {
					cl.Add(arg.String())
				}
				return goja.Undefined()
			})
			_ = clObj.Set("remove", func(c goja.FunctionCall) goja.Value {
				for _, arg := range c.Arguments {
					cl.Remove(arg.String())
				}
				return goja.Undefined()
			})
			_ = clObj.Set("toggle", func(c goja.FunctionCall) goja.Value {
				return b.vm.runtime.ToValue(cl.Toggle(c.Argument(0).String()))
			})
			_ = clObj.Set("contains", func(c goja.FunctionCall) goja.Value {
				return b.vm.runtime.ToValue(cl.Contains(c.Argument(0).String()))
			})

			return b.vm.runtime.ToValue(clObj)
		}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)
}

func (b *docBinder) addDatasetProperty(obj *goja.Object, el *dom.Element) {
	_ = obj.DefineAccessorProperty("dataset",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			ds := el.Dataset()
			proxy := b.vm.runtime.NewDynamicObject(&datasetProxy{ds: ds, rt: b.vm.runtime})
			return b.vm.runtime.ToValue(proxy)
		}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)
}

// wrapNode wraps any dom.Node as a JS object.
func (b *docBinder) wrapNode(n dom.Node) goja.Value {
	switch v := n.(type) {
	case *dom.Element:
		return b.wrapElement(v)
	case *dom.Text:
		return b.wrapTextNode(v)
	case *dom.Document:
		return b.vm.runtime.Get("document")
	case *dom.DocumentFragment:
		return b.wrapFragment(v)
	default:
		return goja.Null()
	}
}

func (b *docBinder) wrapTextNode(tn *dom.Text) goja.Value {
	obj := b.vm.runtime.NewObject()
	_ = obj.DefineAccessorProperty("textContent",
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			return b.vm.runtime.ToValue(tn.TextContent())
		}),
		b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			tn.SetTextContent(call.Argument(0).String())
			return goja.Undefined()
		}),
		goja.FLAG_FALSE, goja.FLAG_TRUE)
	_ = obj.Set("_goNode", tn)
	return b.vm.runtime.ToValue(obj)
}

func (b *docBinder) wrapFragment(frag *dom.DocumentFragment) goja.Value {
	obj := b.vm.runtime.NewObject()
	b.addNodeMethods(obj, frag)
	_ = obj.Set("_goNode", frag)
	return b.vm.runtime.ToValue(obj)
}

// unwrapNode extracts the Go dom.Node from a JS wrapper object.
func (b *docBinder) unwrapNode(v goja.Value) dom.Node {
	obj := v.ToObject(b.vm.runtime)
	goNode := obj.Get("_goNode")
	if goNode != nil && !goja.IsUndefined(goNode) && !goja.IsNull(goNode) {
		if n, ok := goNode.Export().(dom.Node); ok {
			return n
		}
	}
	panic(b.vm.runtime.NewTypeError("argument is not a DOM node"))
}
