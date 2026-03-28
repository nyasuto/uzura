package cdp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/nyasuto/uzura/internal/cdp"
	"github.com/coder/websocket"
)

func startTestServer(t *testing.T, opts ...cdp.ServerOption) *cdp.Server {
	t.Helper()
	allOpts := append([]cdp.ServerOption{cdp.WithAddr(":0")}, opts...)
	s := cdp.NewServer(allOpts...)
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	return s
}

func TestDiscoveryVersion(t *testing.T) {
	s := startTestServer(t, cdp.WithBrowserVersion("Uzura/test"))
	url := fmt.Sprintf("http://%s/json/version", s.Addr())

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /json/version: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var info struct {
		Browser         string `json:"Browser"`
		ProtocolVersion string `json:"Protocol-Version"`
		WebSocketURL    string `json:"webSocketDebuggerUrl"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info.Browser != "Uzura/test" {
		t.Errorf("Browser = %q, want %q", info.Browser, "Uzura/test")
	}
	if info.ProtocolVersion != "1.3" {
		t.Errorf("ProtocolVersion = %q, want %q", info.ProtocolVersion, "1.3")
	}
	if info.WebSocketURL == "" {
		t.Error("WebSocketURL is empty")
	}
}

func TestDiscoveryList(t *testing.T) {
	s := startTestServer(t)
	url := fmt.Sprintf("http://%s/json/list", s.Addr())

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /json/list: %v", err)
	}
	defer resp.Body.Close()

	var targets []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &targets); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != "default" {
		t.Errorf("target ID = %q, want %q", targets[0].ID, "default")
	}
	if targets[0].Type != "page" {
		t.Errorf("target type = %q, want %q", targets[0].Type, "page")
	}
}

func TestDiscoveryListShortPath(t *testing.T) {
	s := startTestServer(t)
	url := fmt.Sprintf("http://%s/json", s.Addr())

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /json: %v", err)
	}
	defer resp.Body.Close()

	var targets []struct {
		ID string `json:"id"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &targets); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
}

func TestDiscoveryProtocol(t *testing.T) {
	s := startTestServer(t)
	url := fmt.Sprintf("http://%s/json/protocol", s.Addr())

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /json/protocol: %v", err)
	}
	defer resp.Body.Close()

	var info struct {
		Domains []struct {
			Name string `json:"name"`
		} `json:"domains"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(info.Domains) == 0 {
		t.Fatal("expected at least one domain")
	}

	names := make(map[string]bool)
	for _, d := range info.Domains {
		names[d.Name] = true
	}
	for _, want := range []string{"Page", "DOM", "Runtime", "Network"} {
		if !names[want] {
			t.Errorf("missing domain %q", want)
		}
	}
}

func dialWS(t *testing.T, ctx context.Context, addr string) *websocket.Conn {
	t.Helper()
	wsURL := fmt.Sprintf("ws://%s/devtools/page/default", addr)
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func sendRPC(t *testing.T, ctx context.Context, conn *websocket.Conn, req interface{}) {
	t.Helper()
	data, _ := json.Marshal(req)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readRPC(t *testing.T, ctx context.Context, conn *websocket.Conn) []byte {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return data
}

func TestWebSocketRPC(t *testing.T) {
	s := startTestServer(t)

	s.Handle("Test.echo", func(params json.RawMessage) (json.RawMessage, error) {
		return params, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Test.echo",
		"params": map[string]string{"hello": "world"},
	})

	respData := readRPC(t, ctx, conn)

	var resp struct {
		ID     int64           `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("response ID = %d, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("result = %v, want {hello: world}", result)
	}
}

func TestWebSocketMethodNotFound(t *testing.T) {
	s := startTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     42,
		"method": "NoSuch.method",
	})

	respData := readRPC(t, ctx, conn)

	var resp struct {
		ID    int64 `json:"id"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 42 {
		t.Errorf("response ID = %d, want 42", resp.ID)
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func TestWebSocketHandlerError(t *testing.T) {
	s := startTestServer(t)

	s.Handle("Test.fail", func(_ json.RawMessage) (json.RawMessage, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     7,
		"method": "Test.fail",
	})

	respData := readRPC(t, ctx, conn)

	var resp struct {
		ID    int64 `json:"id"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("error code = %d, want -32000", resp.Error.Code)
	}
	if resp.Error.Message != "something went wrong" {
		t.Errorf("error message = %q", resp.Error.Message)
	}
}

func TestWebSocketMultipleRequests(t *testing.T) {
	s := startTestServer(t)

	callCount := 0
	s.Handle("Test.count", func(_ json.RawMessage) (json.RawMessage, error) {
		callCount++
		return json.Marshal(map[string]int{"count": callCount})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	for i := 1; i <= 3; i++ {
		sendRPC(t, ctx, conn, map[string]interface{}{
			"id":     i,
			"method": "Test.count",
		})

		respData := readRPC(t, ctx, conn)

		var resp struct {
			ID     int64           `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(respData, &resp); err != nil {
			t.Fatalf("unmarshal %d: %v", i, err)
		}
		if resp.ID != int64(i) {
			t.Errorf("response %d: ID = %d", i, resp.ID)
		}

		var result map[string]int
		_ = json.Unmarshal(resp.Result, &result)
		if result["count"] != i {
			t.Errorf("response %d: count = %d, want %d", i, result["count"], i)
		}
	}
}
