package cdp_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// waitForRequestPaused reads messages until Fetch.requestPaused arrives, returning params.
func waitForRequestPaused(t *testing.T, ctx context.Context, conn *websocket.Conn) (reqID string, params map[string]interface{}) {
	t.Helper()
	for i := 0; i < 15; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if method, _ := msg["method"].(string); method == "Fetch.requestPaused" {
			params = parseParams(t, msg)
			reqID = params["requestId"].(string)
			return reqID, params
		}
	}
	t.Fatal("did not receive Fetch.requestPaused")
	return "", nil
}

func TestFetchContinueRequestWithURLRewrite(t *testing.T) {
	var lastDocPath atomic.Value
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track document paths, skip background resource hint requests.
		if r.URL.Path != "/favicon.ico" && r.Method == http.MethodGet {
			lastDocPath.Store(r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>rewritten</body></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 1, "method": "Fetch.enable",
		"params": map[string]interface{}{"patterns": []map[string]string{{"urlPattern": "*"}}},
	})
	readRPC(t, ctx, conn)

	// Navigate to /original.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 2, "method": "Page.navigate",
		"params": map[string]string{"url": html.URL + "/original"},
	})

	reqID, _ := waitForRequestPaused(t, ctx, conn)

	// Rewrite URL to /rewritten.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 3, "method": "Fetch.continueRequest",
		"params": map[string]interface{}{
			"requestId": reqID,
			"url":       html.URL + "/rewritten",
		},
	})

	// Wait for navigate response.
	for i := 0; i < 20; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			if msg["error"] != nil {
				t.Errorf("navigate error: %v", msg["error"])
			}
			break
		}
	}

	lastPath, _ := lastDocPath.Load().(string)
	if lastPath != "/rewritten" {
		t.Errorf("server received path %q, want /rewritten", lastPath)
	}
}

func TestFetchContinueRequestWithHeaderInjection(t *testing.T) {
	var gotAuth atomic.Value
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only capture headers from the main document request.
		if r.URL.Path == "/" || r.URL.Path == "" {
			gotAuth.Store(r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 1, "method": "Fetch.enable",
		"params": map[string]interface{}{"patterns": []map[string]string{{"urlPattern": "*"}}},
	})
	readRPC(t, ctx, conn)

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 2, "method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	reqID, _ := waitForRequestPaused(t, ctx, conn)

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 3, "method": "Fetch.continueRequest",
		"params": map[string]interface{}{
			"requestId": reqID,
			"headers": []map[string]string{
				{"name": "Authorization", "value": "Bearer token123"},
			},
		},
	})

	for i := 0; i < 20; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			break
		}
	}

	auth, _ := gotAuth.Load().(string)
	if auth != "Bearer token123" {
		t.Errorf("Authorization header = %q, want %q", auth, "Bearer token123")
	}
}

func TestFetchFulfillRequest(t *testing.T) {
	var serverHit atomic.Bool
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only track main document requests, not background resource hints.
		if r.URL.Path == "/" || r.URL.Path == "" {
			serverHit.Store(true)
		}
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 1, "method": "Fetch.enable",
		"params": map[string]interface{}{"patterns": []map[string]string{{"urlPattern": "*"}}},
	})
	readRPC(t, ctx, conn)

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 2, "method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	reqID, _ := waitForRequestPaused(t, ctx, conn)

	mockBody := "<html><body>mocked!</body></html>"
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 3, "method": "Fetch.fulfillRequest",
		"params": map[string]interface{}{
			"requestId":    reqID,
			"responseCode": 200,
			"responseHeaders": []map[string]string{
				{"name": "Content-Type", "value": "text/html"},
			},
			"body": base64.StdEncoding.EncodeToString([]byte(mockBody)),
		},
	})

	for i := 0; i < 20; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			if msg["error"] != nil {
				t.Errorf("navigate error: %v", msg["error"])
			}
			break
		}
	}

	if serverHit.Load() {
		t.Error("HTTP server should NOT have been hit with fulfillRequest")
	}
}

func TestFetchResponseStageGetBodyAndContinue(t *testing.T) {
	const serverBody = `<html><body>original</body></html>`
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("X-Custom", "server-value")
		w.WriteHeader(200)
		w.Write([]byte(serverBody))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable with Response stage pattern.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 1, "method": "Fetch.enable",
		"params": map[string]interface{}{
			"patterns": []map[string]string{
				{"urlPattern": "*", "requestStage": "Response"},
			},
		},
	})
	readRPC(t, ctx, conn)

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 2, "method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	reqID, params := waitForRequestPaused(t, ctx, conn)

	// Verify response stage fields are present.
	if sc, ok := params["responseStatusCode"]; !ok || int(sc.(float64)) != 200 {
		t.Errorf("responseStatusCode = %v, want 200", params["responseStatusCode"])
	}

	// Get response body.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 3, "method": "Fetch.getResponseBody",
		"params": map[string]string{"requestId": reqID},
	})

	var gotBody string
	for i := 0; i < 10; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 3 {
			result := msg["result"].(map[string]interface{})
			bodyB64 := result["body"].(string)
			decoded, _ := base64.StdEncoding.DecodeString(bodyB64)
			gotBody = string(decoded)
			break
		}
	}

	if gotBody != serverBody {
		t.Errorf("getResponseBody = %q, want %q", gotBody, serverBody)
	}

	// Continue response with header override.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 4, "method": "Fetch.continueResponse",
		"params": map[string]interface{}{
			"requestId": reqID,
			"responseHeaders": []map[string]string{
				{"name": "X-Injected", "value": "true"},
			},
		},
	})

	// Wait for navigate to complete.
	for i := 0; i < 20; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			if msg["error"] != nil {
				t.Errorf("navigate error: %v", msg["error"])
			}
			break
		}
	}
}

func TestFetchFulfillAtResponseStage(t *testing.T) {
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>server response</body></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 1, "method": "Fetch.enable",
		"params": map[string]interface{}{
			"patterns": []map[string]string{
				{"urlPattern": "*", "requestStage": "Response"},
			},
		},
	})
	readRPC(t, ctx, conn)

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 2, "method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	reqID, _ := waitForRequestPaused(t, ctx, conn)

	// Fulfill with a completely different response at response stage.
	mockBody := "<html><body>replaced!</body></html>"
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id": 3, "method": "Fetch.fulfillRequest",
		"params": map[string]interface{}{
			"requestId":    reqID,
			"responseCode": 201,
			"responseHeaders": []map[string]string{
				{"name": "Content-Type", "value": "text/html"},
			},
			"body": base64.StdEncoding.EncodeToString([]byte(mockBody)),
		},
	})

	for i := 0; i < 20; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			if msg["error"] != nil {
				t.Errorf("navigate error: %v", msg["error"])
			}
			break
		}
	}
}
