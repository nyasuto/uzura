package js

import (
	"bytes"
	"strings"
	"testing"
)

func TestVMEval(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   interface{}
	}{
		{"integer", "1 + 2", int64(3)},
		{"string", "'hello' + ' world'", "hello world"},
		{"boolean", "true && false", false},
		{"undefined", "undefined", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := New()
			got, err := vm.Eval(tt.script)
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.script, err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v (%T), want %v (%T)", tt.script, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestVMEvalError(t *testing.T) {
	vm := New()
	_, err := vm.Eval("throw new Error('boom')")
	if err == nil {
		t.Fatal("expected error for throw statement")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should contain 'boom', got: %v", err)
	}
}

func TestVMConsoleLog(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`console.log("hello", 42)`)
	if err != nil {
		t.Fatalf("console.log error: %v", err)
	}
	if got := buf.String(); got != "hello 42\n" {
		t.Errorf("console.log output = %q, want %q", got, "hello 42\n")
	}
}

func TestVMConsoleWarn(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`console.warn("warning!")`)
	if err != nil {
		t.Fatalf("console.warn error: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, "warning!") {
		t.Errorf("console.warn output = %q, want to contain 'warning!'", got)
	}
}

func TestVMConsoleError(t *testing.T) {
	var buf bytes.Buffer
	vm := New(WithWriter(&buf))

	_, err := vm.Eval(`console.error("err!")`)
	if err != nil {
		t.Fatalf("console.error error: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, "err!") {
		t.Errorf("console.error output = %q, want to contain 'err!'", got)
	}
}

func TestVMSandboxNoRequire(t *testing.T) {
	vm := New()
	_, err := vm.Eval(`require('fs')`)
	if err == nil {
		t.Fatal("expected error for require() call")
	}
}

func TestVMWindowGlobalThis(t *testing.T) {
	vm := New()

	got, err := vm.Eval(`typeof globalThis`)
	if err != nil {
		t.Fatalf("globalThis error: %v", err)
	}
	if got != "object" {
		t.Errorf("typeof globalThis = %v, want 'object'", got)
	}

	got, err = vm.Eval(`typeof window`)
	if err != nil {
		t.Fatalf("window error: %v", err)
	}
	if got != "object" {
		t.Errorf("typeof window = %v, want 'object'", got)
	}
}

func TestVMReset(t *testing.T) {
	vm := New()

	_, err := vm.Eval(`var x = 42`)
	if err != nil {
		t.Fatalf("set var error: %v", err)
	}

	vm.Reset()

	_, err = vm.Eval(`x`)
	if err == nil {
		t.Fatal("expected error after reset, variable should not exist")
	}
}
