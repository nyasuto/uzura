package js

import (
	"context"
	"io"
	"testing"
	"time"
)

// TestXHR_PageDeadlineFiresError guards against misclassifying a page-level
// (RunEventLoopContext) deadline as this XHR's own timeout. No xhr.timeout
// is set, so a context.DeadlineExceeded surfacing from the page deadline
// must fire onerror, never ontimeout.
func TestXHR_PageDeadlineFiresError(t *testing.T) {
	client := func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(client)
	vm.SetBaseURL("https://example.com/")
	BindXHR(vm)

	// The open+send call must happen once the loop (and its deadline
	// context) is already running — vm.LoopContext() only reflects
	// RunEventLoopContext's ctx while the loop is executing, so scheduling
	// this via setTimeout(0) (a loop task) rather than calling it inline
	// from vm.Eval (which runs before the loop starts) is required for
	// xhrSend to actually observe the page deadline.
	_, err := vm.Eval(`
		var result = {};
		var xhr;
		setTimeout(function() {
			xhr = new XMLHttpRequest();
			xhr.onerror = function() { result.errored = true; result.readyState = xhr.readyState; result.status = xhr.status; };
			xhr.ontimeout = function() { result.timedOut = true; };
			xhr.open("GET", "/slow");
			xhr.send();
		}, 0);
	`)
	if err != nil {
		t.Fatal(err)
	}

	// The page-level deadline fires well before the fake client returns.
	// RunEventLoopContext returns ctx.Err() (DeadlineExceeded) as soon as it
	// observes that — it does not wait for xhrSend's goroutine to deliver
	// its result via completeWith. Tolerate that loop error here.
	deadlineCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_ = vm.RunEventLoopContext(deadlineCtx)

	// xhrSend's request context is a child of the now-expired page context,
	// so it is already canceled too; the fake client's goroutine unblocks
	// almost immediately and calls completeWith. Drain that with a fresh,
	// generously-bounded context so the JS-side handler actually runs
	// before we assert on it.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	if err := vm.RunEventLoopContext(drainCtx); err != nil {
		t.Fatalf("drain RunEventLoopContext: %v", err)
	}

	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"errored":true,"readyState":4,"status":0}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestXHR_AbortInFlight verifies that calling abort() while a request is in
// flight fires onabort exactly once, synchronously drives readyState to
// DONE with status 0, and that the request goroutine's eventual (late)
// completion is swallowed rather than firing onload/onerror a second time.
func TestXHR_AbortInFlight(t *testing.T) {
	client := func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(client)
	vm.SetBaseURL("https://example.com/")
	BindXHR(vm)

	_, err := vm.Eval(`
		var result = {aborted: 0, loaded: false, errored: false};
		var xhr = new XMLHttpRequest();
		xhr.onabort = function() {
			result.aborted++;
			result.readyState = xhr.readyState;
			result.status = xhr.status;
		};
		xhr.onload = function() { result.loaded = true; };
		xhr.onerror = function() { result.errored = true; };
		xhr.open("GET", "/slow");
		xhr.send();
		setTimeout(function() { xhr.abort(); }, 20);
	`)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := vm.RunEventLoopContext(ctx); err != nil {
		t.Fatal(err)
	}

	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"aborted":1,"loaded":false,"errored":false,"readyState":4,"status":0}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
