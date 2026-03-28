package js

import (
	"bytes"
	"testing"
)

func TestSetTimeout(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`setTimeout(function() { console.log("fired"); }, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	vm.RunEventLoop()

	if got := buf.String(); got != "fired\n" {
		t.Errorf("setTimeout output = %q, want %q", got, "fired\n")
	}
}

func TestSetTimeoutOrdering(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`
		setTimeout(function() { console.log("second"); }, 20);
		setTimeout(function() { console.log("first"); }, 10);
	`)
	if err != nil {
		t.Fatal(err)
	}
	vm.RunEventLoop()

	if got := buf.String(); got != "first\nsecond\n" {
		t.Errorf("ordering output = %q, want %q", got, "first\nsecond\n")
	}
}

func TestClearTimeout(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`
		var id = setTimeout(function() { console.log("should not fire"); }, 10);
		clearTimeout(id);
		setTimeout(function() { console.log("ok"); }, 20);
	`)
	if err != nil {
		t.Fatal(err)
	}
	vm.RunEventLoop()

	if got := buf.String(); got != "ok\n" {
		t.Errorf("clearTimeout output = %q, want %q", got, "ok\n")
	}
}

func TestSetInterval(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`
		var count = 0;
		var id = setInterval(function() {
			count++;
			console.log("tick" + count);
			if (count >= 3) clearInterval(id);
		}, 10);
	`)
	if err != nil {
		t.Fatal(err)
	}
	vm.RunEventLoop()

	if got := buf.String(); got != "tick1\ntick2\ntick3\n" {
		t.Errorf("setInterval output = %q, want %q", got, "tick1\ntick2\ntick3\n")
	}
}

func TestNestedSetTimeout(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`
		setTimeout(function() {
			console.log("outer");
			setTimeout(function() {
				console.log("inner");
			}, 0);
		}, 0);
	`)
	if err != nil {
		t.Fatal(err)
	}
	vm.RunEventLoop()

	if got := buf.String(); got != "outer\ninner\n" {
		t.Errorf("nested output = %q, want %q", got, "outer\ninner\n")
	}
}
