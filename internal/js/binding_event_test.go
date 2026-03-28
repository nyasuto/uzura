package js

import (
	"bytes"
	"testing"

	_ "github.com/nyasuto/uzura/internal/html"
)

func TestAddEventListenerAndDispatch(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, makeTestDoc())

	_, err := vm.Eval(`
		var el = document.getElementById("content");
		el.addEventListener("click", function(e) {
			console.log("clicked:" + e.type);
		});
		el.dispatchEvent(new Event("click"));
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "clicked:click\n" {
		t.Errorf("output = %q, want %q", got, "clicked:click\n")
	}
}

func TestRemoveEventListener(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, makeTestDoc())

	_, err := vm.Eval(`
		var el = document.getElementById("content");
		var handler = function(e) { console.log("should not fire"); };
		el.addEventListener("click", handler);
		el.removeEventListener("click", handler);
		el.dispatchEvent(new Event("click"));
	`)
	if err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 0 {
		t.Errorf("handler should not have fired, got: %q", buf.String())
	}
}

func TestEventBubbling(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, makeTestDoc())

	_, err := vm.Eval(`
		var results = [];
		document.body.addEventListener("click", function(e) {
			results.push("body:" + e.target.id);
		});
		var el = document.getElementById("content");
		el.addEventListener("click", function(e) {
			results.push("el:" + e.target.id);
		});
		el.dispatchEvent(new Event("click", { bubbles: true }));
		console.log(results.join(","));
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "el:content,body:content\n" {
		t.Errorf("bubbling output = %q, want %q", got, "el:content,body:content\n")
	}
}

func TestEventCapturing(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, makeTestDoc())

	_, err := vm.Eval(`
		var results = [];
		document.body.addEventListener("click", function(e) {
			results.push("capture:body");
		}, true);
		var el = document.getElementById("content");
		el.addEventListener("click", function(e) {
			results.push("target:el");
		});
		el.dispatchEvent(new Event("click", { bubbles: true }));
		console.log(results.join(","));
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "capture:body,target:el\n" {
		t.Errorf("capture output = %q, want %q", got, "capture:body,target:el\n")
	}
}

func TestEventPreventDefault(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`
		var el = document.getElementById("content");
		var prevented = false;
		el.addEventListener("click", function(e) {
			e.preventDefault();
		});
		var ev = new Event("click", { cancelable: true });
		el.dispatchEvent(ev);
		ev.defaultPrevented;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Errorf("defaultPrevented = %v, want true", got)
	}
}

func TestEventStopPropagation(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, makeTestDoc())

	_, err := vm.Eval(`
		var el = document.getElementById("content");
		el.addEventListener("click", function(e) {
			e.stopPropagation();
			console.log("target");
		});
		document.body.addEventListener("click", function(e) {
			console.log("should not reach");
		});
		el.dispatchEvent(new Event("click", { bubbles: true }));
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "target\n" {
		t.Errorf("stopPropagation output = %q, want %q", got, "target\n")
	}
}

func TestDocumentAndWindowEventTarget(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))
	BindDocument(vm, makeTestDoc())

	_, err := vm.Eval(`
		document.addEventListener("custom", function(e) {
			console.log("doc:" + e.type);
		});
		document.dispatchEvent(new Event("custom"));
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "doc:custom\n" {
		t.Errorf("document event output = %q, want %q", got, "doc:custom\n")
	}
}
