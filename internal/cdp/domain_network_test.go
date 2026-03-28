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
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

func startNetworkServer(t *testing.T, handler http.Handler) (srv *cdp.Server, htmlSrv *httptest.Server) {
	t.Helper()

	htmlSrv = httptest.NewServer(handler)
	t.Cleanup(htmlSrv.Close)

	nd := cdp.NewNetworkDomain(nil)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{
		Fetcher:         fetcher,
		NetworkObserver: nd.Observer(),
	})
	nd.SetPage(p)

	pageDomain := cdp.NewPageDomain(p)

	s := cdp.NewServer(cdp.WithAddr(":0"))
	pageDomain.Register(s)
	nd.Register(s)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	return s, htmlSrv
}

func TestNetworkEnable(t *testing.T) {
	s, _ := startNetworkServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Network.enable",
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
	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
}

func TestNetworkEventsOnNavigate(t *testing.T) {
	const body = `<!DOCTYPE html><html><head><title>Net Test</title></head><body><p>Hello</p></body></html>`
	s, html := startNetworkServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte(body))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable Network first.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Network.enable",
	})
	readRPC(t, ctx, conn) // enable response

	// Navigate.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// Collect messages: 3 network events (async) + 1 response + 9 page lifecycle events.
	messages := readAllMessages(t, ctx, conn, 12)

	var (
		reqWillBeSent bool
		respReceived  bool
		loadFinished  bool
		navigateResp  bool
		requestID     string
	)

	for _, msg := range messages {
		method, _ := msg["method"].(string)
		switch method {
		case "Network.requestWillBeSent":
			reqWillBeSent = true
			params := parseParams(t, msg)
			requestID = params["requestId"].(string)
			if requestID == "" {
				t.Error("requestWillBeSent: missing requestId")
			}
			req := params["request"].(map[string]interface{})
			if req["url"] != html.URL {
				t.Errorf("request url = %v, want %v", req["url"], html.URL)
			}
			if req["method"] != "GET" {
				t.Errorf("request method = %v, want GET", req["method"])
			}

		case "Network.responseReceived":
			respReceived = true
			params := parseParams(t, msg)
			resp := params["response"].(map[string]interface{})
			if int(resp["status"].(float64)) != 200 {
				t.Errorf("response status = %v, want 200", resp["status"])
			}
			if resp["mimeType"] != "text/html" {
				t.Errorf("mimeType = %v, want text/html", resp["mimeType"])
			}

		case "Network.loadingFinished":
			loadFinished = true
			params := parseParams(t, msg)
			length := int64(params["encodedDataLength"].(float64))
			if length != int64(len(body)) {
				t.Errorf("encodedDataLength = %d, want %d", length, len(body))
			}

		case "": // Response (no method field).
			if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
				navigateResp = true
			}
		}
	}

	if !reqWillBeSent {
		t.Error("missing Network.requestWillBeSent event")
	}
	if !respReceived {
		t.Error("missing Network.responseReceived event")
	}
	if !loadFinished {
		t.Error("missing Network.loadingFinished event")
	}
	if !navigateResp {
		t.Error("missing Page.navigate response")
	}
}

func TestNetworkGetResponseBody(t *testing.T) {
	const htmlBody = `<html><body>response body test</body></html>`
	s, html := startNetworkServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlBody))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable network.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Network.enable",
	})
	readRPC(t, ctx, conn)

	// Navigate.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// Collect all messages to find the requestId.
	messages := readAllMessages(t, ctx, conn, 12)

	var requestID string
	for _, msg := range messages {
		if method, _ := msg["method"].(string); method == "Network.requestWillBeSent" {
			params := parseParams(t, msg)
			requestID = params["requestId"].(string)
			break
		}
	}
	if requestID == "" {
		t.Fatal("could not find requestId from Network.requestWillBeSent")
	}

	// Get response body.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     3,
		"method": "Network.getResponseBody",
		"params": map[string]string{"requestId": requestID},
	})

	respData := readRPC(t, ctx, conn)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			Body          string `json:"body"`
			Base64Encoded bool   `json:"base64Encoded"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	if resp.Result.Body != htmlBody {
		t.Errorf("body = %q, want %q", resp.Result.Body, htmlBody)
	}
	if resp.Result.Base64Encoded {
		t.Error("expected base64Encoded = false")
	}
}

func TestNetworkLoadingFailed(t *testing.T) {
	s, _ := startNetworkServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable network.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Network.enable",
	})
	readRPC(t, ctx, conn)

	// Navigate to an unreachable URL.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": "http://127.0.0.1:1/unreachable"},
	})

	// Collect: requestWillBeSent + loadingFailed + navigate response (with errorText).
	messages := readAllMessages(t, ctx, conn, 3)

	var loadFailed bool
	for _, msg := range messages {
		if method, _ := msg["method"].(string); method == "Network.loadingFailed" {
			loadFailed = true
			params := parseParams(t, msg)
			if params["errorText"] == nil || params["errorText"].(string) == "" {
				t.Error("loadingFailed: missing errorText")
			}
		}
	}
	if !loadFailed {
		t.Error("missing Network.loadingFailed event")
	}
}

func TestNetworkRequestIDManagement(t *testing.T) {
	s, html := startNetworkServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Network.enable",
	})
	readRPC(t, ctx, conn)

	// First navigation.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})
	msgs1 := readAllMessages(t, ctx, conn, 12)

	// Second navigation.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     3,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})
	msgs2 := readAllMessages(t, ctx, conn, 12)

	id1 := extractRequestID(t, msgs1)
	id2 := extractRequestID(t, msgs2)

	if id1 == id2 {
		t.Errorf("request IDs should be unique across navigations, got %q both times", id1)
	}

	// Verify all events for the same navigation share the same requestId.
	for _, msg := range msgs1 {
		method, _ := msg["method"].(string)
		if method == "" {
			continue
		}
		params := parseParams(t, msg)
		if rid, ok := params["requestId"]; ok && rid != id1 {
			t.Errorf("event %s has requestId %v, want %v", method, rid, id1)
		}
	}
}

func TestNetworkEventTimeOrder(t *testing.T) {
	s, html := startNetworkServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Network.enable",
	})
	readRPC(t, ctx, conn)

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	messages := readAllMessages(t, ctx, conn, 12)

	// Extract timestamps in order of network events.
	var timestamps []float64
	order := []string{"Network.requestWillBeSent", "Network.responseReceived", "Network.loadingFinished"}

	for _, wantMethod := range order {
		for _, msg := range messages {
			if method, _ := msg["method"].(string); method == wantMethod {
				params := parseParams(t, msg)
				ts := params["timestamp"].(float64)
				timestamps = append(timestamps, ts)
				break
			}
		}
	}

	if len(timestamps) != 3 {
		t.Fatalf("expected 3 timestamps, got %d", len(timestamps))
	}

	for i := 1; i < len(timestamps); i++ {
		if timestamps[i] < timestamps[i-1] {
			t.Errorf("timestamp[%d] (%f) < timestamp[%d] (%f): events out of order",
				i, timestamps[i], i-1, timestamps[i-1])
		}
	}
}

// --- helpers ---

func readAllMessages(t *testing.T, ctx context.Context, conn *websocket.Conn, count int) []map[string]interface{} {
	t.Helper()
	msgs := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal message %d: %v", i, err)
		}
		msgs[i] = msg
	}
	return msgs
}

func parseParams(t *testing.T, msg map[string]interface{}) map[string]interface{} {
	t.Helper()
	params, ok := msg["params"].(map[string]interface{})
	if !ok {
		t.Fatal("message has no params")
	}
	return params
}

func extractRequestID(t *testing.T, msgs []map[string]interface{}) string {
	t.Helper()
	for _, msg := range msgs {
		if method, _ := msg["method"].(string); method == "Network.requestWillBeSent" {
			params := parseParams(t, msg)
			return params["requestId"].(string)
		}
	}
	t.Fatal("no Network.requestWillBeSent found")
	return ""
}
