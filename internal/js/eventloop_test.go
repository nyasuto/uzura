package js

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
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

func TestEventLoop_TaskQueue(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	var order []string
	vm.loop.enqueueTask(func() { order = append(order, "task1") })
	vm.loop.enqueueTask(func() { order = append(order, "task2") })
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatalf("RunEventLoopContext: %v", err)
	}
	if len(order) != 2 || order[0] != "task1" || order[1] != "task2" {
		t.Errorf("order = %v, want [task1 task2]", order)
	}
}

func TestEventLoop_WaitsForPending(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	var got bool
	vm.loop.addPending()
	go func() {
		time.Sleep(50 * time.Millisecond)
		vm.loop.enqueueTask(func() { got = true })
		vm.loop.donePending()
	}()
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatalf("RunEventLoopContext: %v", err)
	}
	if !got {
		t.Error("loop exited before pending async work completed")
	}
}

func TestEventLoop_ContextDeadline(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	vm.loop.addPending() // 誰も donePending しない = ハングのシミュレーション
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := vm.RunEventLoopContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}

func TestEventLoop_TasksAndTimersInterleave(t *testing.T) {
	// 検証意図: タスクは「まだ期限が来ていないタイマー」より先に処理される
	vm := New(WithWriter(io.Discard))
	_, _ = vm.Eval(`var __order__ = []; setTimeout(function() { __order__.push("timer"); }, 20);`)
	vm.loop.enqueueTask(func() {
		_, _ = vm.Eval(`__order__.push("task")`)
	})
	_ = vm.RunEventLoopContext(context.Background())
	got, _ := vm.Eval(`JSON.stringify(__order__)`)
	if got != `["task","timer"]` {
		t.Errorf("order = %v, want [task timer]", got)
	}
}
