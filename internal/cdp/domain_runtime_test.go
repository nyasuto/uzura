package cdp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/nyasuto/uzura/internal/cdp"
	"github.com/nyasuto/uzura/internal/js"
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

const runtimeTestHTML = `<!DOCTYPE html><html><head><title>Runtime Test</title></head>` +
	`<body><div id="app">Hello</div></body></html>`

func startRuntimeServer(t *testing.T) (srv *cdp.Server, htmlSrv *httptest.Server) {
	t.Helper()

	html := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(runtimeTestHTML))
	}))
	t.Cleanup(html.Close)

	rd := cdp.NewRuntimeDomain(nil)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{
		Fetcher:   fetcher,
		VMOptions: []js.Option{js.WithConsoleCallback(rd.ConsoleCallback())},
	})
	rd.SetPage(p)

	s := cdp.NewServer(cdp.WithAddr(":0"))
	pageDomain := cdp.NewPageDomain(p)
	pageDomain.Register(s)
	domDomain := cdp.NewDOMDomain(p)
	domDomain.Register(s)
	rd.Register(s)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	return s, html
}

// startRuntimeServerWithConsole sets up a server where console callback is wired to the runtime domain.
func startRuntimeServerWithConsole(t *testing.T) (srv *cdp.Server, htmlSrv *httptest.Server, rd *cdp.RuntimeDomain) {
	t.Helper()

	html := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(runtimeTestHTML))
	}))
	t.Cleanup(html.Close)

	rd = cdp.NewRuntimeDomain(nil)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{
		Fetcher:   fetcher,
		VMOptions: []js.Option{js.WithConsoleCallback(rd.ConsoleCallback())},
	})
	rd.SetPage(p)

	s := cdp.NewServer(cdp.WithAddr(":0"))
	pageDomain := cdp.NewPageDomain(p)
	pageDomain.Register(s)
	rd.Register(s)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	return s, html, rd
}

func TestRuntimeEnable(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.enable", nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestRuntimeEvaluateNumber(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "1 + 2",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Result.Type != "number" {
		t.Errorf("type = %q, want number", result.Result.Type)
	}
	// goja exports int64 for integer results
	val, ok := result.Result.Value.(float64)
	if !ok {
		t.Fatalf("value type = %T, want float64 (from JSON)", result.Result.Value)
	}
	if val != 3 {
		t.Errorf("value = %v, want 3", val)
	}
}

func TestRuntimeEvaluateString(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "'hello' + ' world'",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.Result.Type != "string" {
		t.Errorf("type = %q, want string", result.Result.Type)
	}
	if result.Result.Value != "hello world" {
		t.Errorf("value = %v, want 'hello world'", result.Result.Value)
	}
}

func TestRuntimeEvaluateBoolean(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "true",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.Result.Type != "boolean" {
		t.Errorf("type = %q, want boolean", result.Result.Type)
	}
	if result.Result.Value != true {
		t.Errorf("value = %v, want true", result.Result.Value)
	}
}

func TestRuntimeEvaluateUndefined(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "undefined",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.Result.Type != "undefined" {
		t.Errorf("type = %q, want undefined", result.Result.Type)
	}
}

func TestRuntimeEvaluateObject(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "({a: 1, b: 2})",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.Result.Type != "object" {
		t.Errorf("type = %q, want object", result.Result.Type)
	}
	if result.Result.ObjectID == "" {
		t.Error("objectId should be assigned for objects")
	}
}

func TestRuntimeEvaluateArray(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "[1, 2, 3]",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.Result.Type != "object" {
		t.Errorf("type = %q, want object", result.Result.Type)
	}
	if result.Result.Subtype != "array" {
		t.Errorf("subtype = %q, want array", result.Result.Subtype)
	}
	if result.Result.ObjectID == "" {
		t.Error("objectId should be assigned for arrays")
	}
}

func TestRuntimeEvaluateError(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable first to capture exceptionThrown event.
	callRPC(t, ctx, conn, 1, "Runtime.enable", nil)

	// Send evaluate with syntax error.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{"expression": "throw new Error('test error')"},
	})

	// Read response.
	respData := readRPC(t, ctx, conn)
	var resp rpcResponse
	json.Unmarshal(respData, &resp)

	// The RPC itself should succeed (no error field), but result should have exceptionDetails.
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	var result struct {
		Result           cdp.RemoteObject       `json:"result"`
		ExceptionDetails map[string]interface{} `json:"exceptionDetails"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.ExceptionDetails == nil {
		t.Fatal("expected exceptionDetails for thrown error")
	}

	// Should also receive Runtime.exceptionThrown event.
	evtData := readRPC(t, ctx, conn)
	var evt struct {
		Method string `json:"method"`
	}
	json.Unmarshal(evtData, &evt)
	if evt.Method != "Runtime.exceptionThrown" {
		t.Errorf("event method = %q, want Runtime.exceptionThrown", evt.Method)
	}
}

func TestRuntimeEvaluateWithDOM(t *testing.T) {
	s, html := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	navigateAndDrain(t, ctx, conn, html.URL)

	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "document.title",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &result)
	if result.Result.Type != "string" {
		t.Errorf("type = %q, want string", result.Result.Type)
	}
	if result.Result.Value != "Runtime Test" {
		t.Errorf("value = %v, want 'Runtime Test'", result.Result.Value)
	}
}

func TestRuntimeCallFunctionOn(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// First evaluate to create an object.
	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "({x: 10, y: 20})",
	})
	var evalResult struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &evalResult)
	objID := evalResult.Result.ObjectID
	if objID == "" {
		t.Fatal("expected objectId from evaluate")
	}

	// Call function on that object.
	resp = callRPC(t, ctx, conn, 2, "Runtime.callFunctionOn", map[string]interface{}{
		"objectId":            objID,
		"functionDeclaration": "function() { return this.x + this.y; }",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var callResult struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &callResult)
	if callResult.Result.Type != "number" {
		t.Errorf("type = %q, want number", callResult.Result.Type)
	}
	val, _ := callResult.Result.Value.(float64)
	if val != 30 {
		t.Errorf("value = %v, want 30", val)
	}
}

func TestRuntimeCallFunctionOnWithArgs(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Evaluate to create object.
	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "({base: 100})",
	})
	var evalResult struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &evalResult)
	objID := evalResult.Result.ObjectID

	// Call with value arguments.
	resp = callRPC(t, ctx, conn, 2, "Runtime.callFunctionOn", map[string]interface{}{
		"objectId":            objID,
		"functionDeclaration": "function(a, b) { return this.base + a + b; }",
		"arguments": []map[string]interface{}{
			{"value": 5},
			{"value": 10},
		},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var callResult struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &callResult)
	val, _ := callResult.Result.Value.(float64)
	if val != 115 {
		t.Errorf("value = %v, want 115", val)
	}
}

func TestRuntimeCallFunctionOnError(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Evaluate to create object.
	resp := callRPC(t, ctx, conn, 1, "Runtime.evaluate", map[string]interface{}{
		"expression": "({x: 1})",
	})
	var evalResult struct {
		Result cdp.RemoteObject `json:"result"`
	}
	json.Unmarshal(resp.Result, &evalResult)
	objID := evalResult.Result.ObjectID

	// Call function that throws.
	resp = callRPC(t, ctx, conn, 2, "Runtime.callFunctionOn", map[string]interface{}{
		"objectId":            objID,
		"functionDeclaration": "function() { throw new Error('call error'); }",
	})
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	var callResult struct {
		ExceptionDetails map[string]interface{} `json:"exceptionDetails"`
	}
	json.Unmarshal(resp.Result, &callResult)
	if callResult.ExceptionDetails == nil {
		t.Error("expected exceptionDetails for thrown error")
	}
}

func TestRuntimeCallFunctionOnInvalidObjectID(t *testing.T) {
	s, _ := startRuntimeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	resp := callRPC(t, ctx, conn, 1, "Runtime.callFunctionOn", map[string]interface{}{
		"objectId":            "nonexistent-id",
		"functionDeclaration": "function() { return 1; }",
	})
	if resp.Error == nil {
		t.Fatal("expected error for invalid objectId")
	}
}

func TestRuntimeConsoleAPICalled(t *testing.T) {
	s, _, rd := startRuntimeServerWithConsole(t)
	_ = rd
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable Runtime to set up session for events.
	callRPC(t, ctx, conn, 1, "Runtime.enable", nil)

	// Evaluate console.log.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{"expression": "console.log('hello', 42)"},
	})

	// Console event and evaluate response may arrive in either order.
	// Read both and find the console event.
	var consoleEvt struct {
		Method string `json:"method"`
		Params struct {
			Type string             `json:"type"`
			Args []cdp.RemoteObject `json:"args"`
		} `json:"params"`
	}
	found := false
	for i := 0; i < 2; i++ {
		data := readRPC(t, ctx, conn)
		var msg struct {
			Method string `json:"method"`
		}
		json.Unmarshal(data, &msg)
		if msg.Method == "Runtime.consoleAPICalled" {
			json.Unmarshal(data, &consoleEvt)
			found = true
		}
	}
	if !found {
		t.Fatal("did not receive Runtime.consoleAPICalled event")
	}
	if consoleEvt.Params.Type != "log" {
		t.Errorf("console type = %q, want log", consoleEvt.Params.Type)
	}
	if len(consoleEvt.Params.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(consoleEvt.Params.Args))
	}
	if consoleEvt.Params.Args[0].Type != "string" {
		t.Errorf("arg[0] type = %q, want string", consoleEvt.Params.Args[0].Type)
	}
	if consoleEvt.Params.Args[1].Type != "number" {
		t.Errorf("arg[1] type = %q, want number", consoleEvt.Params.Args[1].Type)
	}
}
