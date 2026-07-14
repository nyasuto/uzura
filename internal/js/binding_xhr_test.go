package js

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestXHR_BasicGet(t *testing.T) {
	// open→send→DONE で responseText が取れ、readystatechange が
	// OPENED(1)→HEADERS_RECEIVED(2)→LOADING(3)→DONE(4) の順で発火する
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/api": {Status: 200, StatusText: "OK",
			Headers: http.Header{"Content-Type": []string{"text/plain"}},
			Body:    []byte("hello"), FinalURL: "https://example.com/api"},
	})
	BindXHR(vm)
	_, err := vm.Eval(`
		var states = [], text = "", loaded = false;
		var xhr = new XMLHttpRequest();
		xhr.onreadystatechange = function() { states.push(xhr.readyState); };
		xhr.onload = function() { loaded = true; text = xhr.responseText; };
		xhr.open("GET", "/api");
		xhr.send();
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = vm.RunEventLoopContext(context.Background())
	got, _ := vm.Eval(`JSON.stringify({states: states, text: text, loaded: loaded, status: xhr.status})`)
	want := `{"states":[1,2,3,4],"text":"hello","loaded":true,"status":200}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestXHR_JSONResponseType(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/api": {Status: 200, StatusText: "OK",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{"title":"hello"}`), FinalURL: "https://example.com/api"},
	})
	BindXHR(vm)
	_, err := vm.Eval(`
		var result = {};
		var xhr = new XMLHttpRequest();
		xhr.responseType = "json";
		xhr.onload = function() { result.title = xhr.response.title; };
		xhr.open("GET", "/api");
		xhr.send();
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"title":"hello"}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestXHR_NetworkError(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{})
	BindXHR(vm)
	_, err := vm.Eval(`
		var result = {};
		var xhr = new XMLHttpRequest();
		xhr.onerror = function() { result.errored = true; result.readyState = xhr.readyState; result.status = xhr.status; };
		xhr.open("GET", "/nope");
		xhr.send();
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`JSON.stringify(result)`)
	want := `{"errored":true,"readyState":4,"status":0}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestXHR_SyncThrows(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{})
	BindXHR(vm)
	_, err := vm.Eval(`
		var xhr = new XMLHttpRequest();
		xhr.open("GET", "/api", false);
	`)
	if err == nil {
		t.Fatal("expected error for synchronous open(), got nil")
	}
}

func TestXHR_PostBodyAndHeaders(t *testing.T) {
	var gotReq HTTPRequest
	client := func(_ context.Context, req HTTPRequest) (*HTTPResponse, error) {
		gotReq = req
		return &HTTPResponse{Status: 200, StatusText: "OK", Body: []byte("ok"), FinalURL: req.URL}, nil
	}
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(client)
	vm.SetBaseURL("https://example.com/")
	BindXHR(vm)

	_, err := vm.Eval(`
		var xhr = new XMLHttpRequest();
		xhr.open("POST", "/post");
		xhr.setRequestHeader("X-Token", "t1");
		xhr.send("payload");
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

func TestXHR_GetAllResponseHeaders(t *testing.T) {
	vm := newFetchVM(t, map[string]*HTTPResponse{
		"https://example.com/api": {Status: 200, StatusText: "OK",
			Headers: http.Header{"Content-Type": []string{"text/plain"}},
			Body:    []byte("hello"), FinalURL: "https://example.com/api"},
	})
	BindXHR(vm)
	_, err := vm.Eval(`
		var headers = "";
		var xhr = new XMLHttpRequest();
		xhr.onload = function() { headers = xhr.getAllResponseHeaders(); };
		xhr.open("GET", "/api");
		xhr.send();
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RunEventLoopContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := vm.Eval(`headers`)
	want := "content-type: text/plain\r\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestXHR_Timeout(t *testing.T) {
	client := func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
		select {
		case <-time.After(2 * time.Second):
			return &HTTPResponse{Status: 200, StatusText: "OK", Body: []byte("late"), FinalURL: req.URL}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	vm := New(WithWriter(io.Discard))
	vm.SetHTTPClient(client)
	vm.SetBaseURL("https://example.com/")
	BindXHR(vm)

	_, err := vm.Eval(`
		var result = {};
		var xhr = new XMLHttpRequest();
		xhr.timeout = 20;
		xhr.ontimeout = function() { result.timedOut = true; result.readyState = xhr.readyState; result.status = xhr.status; };
		xhr.open("GET", "/slow");
		xhr.send();
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
	want := `{"timedOut":true,"readyState":4,"status":0}`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
