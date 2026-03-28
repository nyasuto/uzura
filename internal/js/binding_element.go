package js

import (
	"strconv"

	"github.com/dop251/goja"
	"github.com/nyasuto/uzura/internal/dom"
)

func (b *docBinder) wrapElement(el *dom.Element) goja.Value {
	obj := b.vm.runtime.NewObject()

	_ = obj.DefineAccessorProperty("tagName", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.TagName())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("id", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.Id())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("className", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.ClassName())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("textContent", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(el.TextContent())
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("innerHTML", b.vm.runtime.ToValue(func(call goja.FunctionCall) goja.Value {
		return b.vm.runtime.ToValue(dom.InnerHTML(el))
	}), nil, goja.FLAG_FALSE, goja.FLAG_TRUE)

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
