package js

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// fakeClient returns an HTTPClient serving canned responses keyed by URL.
func fakeClient(responses map[string]*HTTPResponse) HTTPClient {
	return func(_ context.Context, req HTTPRequest) (*HTTPResponse, error) {
		if resp, ok := responses[req.URL]; ok {
			return resp, nil
		}
		return nil, fmt.Errorf("no canned response for %s", req.URL)
	}
}

func newFetchVM(t *testing.T, responses map[string]*HTTPResponse) *VM {
	t.Helper()
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(fakeClient(responses))
	vm.SetBaseURL("https://example.com/")
	BindFetch(vm)
	return vm
}

func TestFetch_Basic(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/api": {
			Status: 200, StatusText: "OK",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{"title":"hello"}`), FinalURL: "https://example.com/api",
		},
	})
	_, err := vm.Eval(`
		var result = {};
		fetch("/api").then(function(res) {
			result.ok = res.ok;
			result.status = res.status;
			result.ctype = res.headers.get("content-type");
			return res.json();
		}).then(function(data) { result.title = data.title; });
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"ok":true,"status":200,"ctype":"application/json","title":"hello"}`
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
}

func TestFetch_HTTPError(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/missing": {
			Status: 404, StatusText: "Not Found",
			Headers: http.Header{}, Body: []byte("nope"), FinalURL: "https://example.com/missing",
		},
	})
	_, err := vm.Eval(`
		var result = {};
		fetch("/missing").then(function(res) {
			result.ok = res.ok;
			result.status = res.status;
			result.statusText = res.statusText;
		}, function() {
			result.rejected = true;
		});
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify({ok: result.ok, status: result.status, statusText: result.statusText, rejected: result.rejected || false})`)
	want := `{"ok":false,"status":404,"statusText":"Not Found","rejected":false}`
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
}

func TestFetch_NetworkError(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{})
	_, err := vm.Eval(`
		var result = {};
		fetch("/nope").then(function(res) {
			result.resolved = true;
		}, function(err) {
			result.caught = true;
			result.name = err.name;
		});
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"caught":true,"name":"TypeError"}`
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
}

func TestFetch_Concurrent(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/a": {Status: 200, StatusText: "OK", Body: []byte("a"), FinalURL: "https://example.com/a"},
		"https://example.com/b": {Status: 200, StatusText: "OK", Body: []byte("b"), FinalURL: "https://example.com/b"},
		"https://example.com/c": {Status: 200, StatusText: "OK", Body: []byte("c"), FinalURL: "https://example.com/c"},
	})
	_, err := vm.Eval(`
		var result = null;
		Promise.all([fetch("/a"), fetch("/b"), fetch("/c")]).then(function(responses) {
			return Promise.all(responses.map(function(r) { return r.text(); }));
		}).then(function(texts) { result = texts; });
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `["a","b","c"]`
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
}

func TestFetch_TextAndArrayBuffer(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/data": {Status: 200, StatusText: "OK", Body: []byte("hello"), FinalURL: "https://example.com/data"},
	})
	// Sequence text() and arrayBuffer() off the same fetch to keep result
	// property insertion order deterministic (they'd otherwise race if
	// driven from two independent fetch() calls).
	_, err := vm.Eval(`
		var result = {};
		fetch("/data").then(function(res) {
			return res.text().then(function(t) {
				result.text = t;
				return res.arrayBuffer();
			});
		}).then(function(buf) { result.len = buf.byteLength; });
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"text":"hello","len":5}`
	if got != want {
		t.Errorf("result = %s, want %s", got, want)
	}
}

func TestFetch_PostBody(t *testing.T) {
	var gotReq HTTPRequest
	client := func(_ context.Context, req HTTPRequest) (*HTTPResponse, error) {
		gotReq = req
		return &HTTPResponse{Status: 200, StatusText: "OK", Body: []byte("ok"), FinalURL: req.URL}, nil
	}
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(client)
	vm.SetBaseURL("https://example.com/")
	BindFetch(vm)

	_, err := vm.Eval(`
		fetch("/post", {method: "POST", headers: {"X-Token": "t1"}, body: "payload"});
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if gotReq.Method != "POST" {
		t.Errorf("Method = %q, want POST", gotReq.Method)
	}
	if gotReq.Headers["X-Token"] != "t1" {
		t.Errorf("Headers[X-Token] = %q, want t1", gotReq.Headers["X-Token"])
	}
	if string(gotReq.Body) != "payload" {
		t.Errorf("Body = %q, want payload", gotReq.Body)
	}
}
