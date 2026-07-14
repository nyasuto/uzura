package js

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestAbortController_AbortsFetch(t *testing.T) {
	block := make(chan struct{})
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(func(ctx context.Context, _ HTTPRequest) (*HTTPResponse, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-block:
			return &HTTPResponse{Status: 200}, nil
		}
	})
	vm.SetBaseURL("https://example.com/")
	BindFetch(vm)
	BindAbort(vm)
	defer close(block)

	_, err := vm.Eval(`
		var result = "";
		var ac = new AbortController();
		fetch("/slow", {signal: ac.signal}).catch(function(e) { result = e.name; });
		setTimeout(function() { ac.abort(); }, 20);
	`)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := vm.RunEventLoopContext(ctx); err != nil {
		t.Fatalf("loop should settle after abort, got %v", err)
	}
	got, _ := vm.Eval(`result`)
	if got != "AbortError" {
		t.Errorf("result = %v, want AbortError", got)
	}
}

func TestAbortSignal_AbortedFlag(t *testing.T) {
	vm := New(WithWriter(io.Discard))
	BindAbort(vm)
	got, err := vm.Eval(`
		var ac = new AbortController();
		var before = ac.signal.aborted;
		ac.abort();
		JSON.stringify([before, ac.signal.aborted]);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `[false,true]` {
		t.Errorf("got %v, want [false,true]", got)
	}
}
