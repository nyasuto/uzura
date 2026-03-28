package js

import (
	"io"
	"testing"
)

func BenchmarkEvalSimple(b *testing.B) {
	vm := New(WithWriter(io.Discard))
	b.ResetTimer()
	for b.Loop() {
		_, _ = vm.Eval("1 + 2 + 3")
	}
}

func BenchmarkEvalDOMQuery(b *testing.B) {
	vm := New(WithWriter(io.Discard))
	BindDocument(vm, makeTestDoc())
	b.ResetTimer()
	for b.Loop() {
		_, _ = vm.Eval(`document.getElementById("content").textContent`)
	}
}

func BenchmarkEvalDOMMutation(b *testing.B) {
	vm := New(WithWriter(io.Discard))
	BindDocument(vm, makeTestDoc())
	b.ResetTimer()
	for b.Loop() {
		_, _ = vm.Eval(`
			var el = document.createElement("div");
			el.textContent = "bench";
			document.body.appendChild(el);
			document.body.removeChild(el);
		`)
	}
}

func BenchmarkExecuteScripts(b *testing.B) {
	vm := New(WithWriter(io.Discard))
	doc := makeTestDoc()
	BindDocument(vm, doc)
	b.ResetTimer()
	for b.Loop() {
		_, _ = vm.Eval(`
			var x = 0;
			for (var i = 0; i < 100; i++) { x += i; }
			x;
		`)
	}
}
