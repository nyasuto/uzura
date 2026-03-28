// Package js provides a sandboxed JavaScript execution environment
// using the goja pure-Go engine.
package js

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dop251/goja"
)

// ConsoleCallback is called when a console method is invoked.
// The method is "log", "warn", "error", or "info". Args are the exported values.
type ConsoleCallback func(method string, args []interface{})

// VM wraps a goja runtime with sandboxing and browser globals.
type VM struct {
	runtime         *goja.Runtime
	writer          io.Writer
	loop            *eventLoop
	consoleCallback ConsoleCallback
}

// Option configures a VM.
type Option func(*VM)

// WithWriter sets the output writer for console methods.
func WithWriter(w io.Writer) Option {
	return func(vm *VM) {
		vm.writer = w
	}
}

// WithConsoleCallback sets a callback invoked on each console method call.
func WithConsoleCallback(cb ConsoleCallback) Option {
	return func(vm *VM) {
		vm.consoleCallback = cb
	}
}

// New creates a new sandboxed JavaScript VM.
func New(opts ...Option) *VM {
	vm := &VM{
		writer: os.Stdout,
	}
	for _, opt := range opts {
		opt(vm)
	}
	vm.init()
	return vm
}

func (vm *VM) init() {
	vm.runtime = goja.New()
	vm.setupGlobals()
	vm.setupConsole()
	vm.setupTimers()
}

func (vm *VM) setupGlobals() {
	global := vm.runtime.GlobalObject()

	_ = vm.runtime.Set("window", global)
}

func (vm *VM) setupConsole() {
	console := vm.runtime.NewObject()
	_ = console.Set("log", vm.makeLogFunc("", "log"))
	_ = console.Set("warn", vm.makeLogFunc("WARN: ", "warning"))
	_ = console.Set("error", vm.makeLogFunc("ERROR: ", "error"))
	_ = console.Set("info", vm.makeLogFunc("", "info"))
	_ = vm.runtime.Set("console", console)
}

func (vm *VM) makeLogFunc(prefix, method string) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		exported := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			exported[i] = arg.Export()
			parts[i] = fmt.Sprintf("%v", exported[i])
		}
		fmt.Fprintf(vm.writer, "%s%s\n", prefix, strings.Join(parts, " "))
		if vm.consoleCallback != nil {
			vm.consoleCallback(method, exported)
		}
		return goja.Undefined()
	}
}

// Eval executes JavaScript source code and returns the result.
// The returned value is the Go export of the goja result (nil for undefined).
func (vm *VM) Eval(src string) (interface{}, error) {
	v, err := vm.runtime.RunString(src)
	if err != nil {
		return nil, err
	}
	exported := v.Export()
	return exported, nil
}

// Reset discards the current runtime and creates a fresh sandboxed VM.
func (vm *VM) Reset() {
	vm.init()
}

// Runtime returns the underlying goja runtime for advanced binding.
func (vm *VM) Runtime() *goja.Runtime {
	return vm.runtime
}
