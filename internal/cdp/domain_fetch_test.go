package cdp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/nyasuto/uzura/internal/cdp"
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

// startFetchServer creates a CDP server with Fetch + Page + Network domains wired up.
func startFetchServer(t *testing.T, handler http.Handler) (srv *cdp.Server, htmlSrv *httptest.Server) {
	t.Helper()

	htmlSrv = httptest.NewServer(handler)
	t.Cleanup(htmlSrv.Close)

	nd := cdp.NewNetworkDomain(nil)
	fd := cdp.NewFetchDomain(nil)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{
		Fetcher:            fetcher,
		NetworkObserver:    nd.Observer(),
		RequestInterceptor: fd.Interceptor(),
	})
	nd.SetPage(p)
	fd.SetPage(p)

	pageDomain := cdp.NewPageDomain(p)

	s := cdp.NewServer(cdp.WithAddr(":0"))
	pageDomain.Register(s)
	nd.Register(s)
	fd.Register(s)

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

func TestFetchEnableDisable(t *testing.T) {
	s, _ := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Fetch.enable",
		"params": map[string]interface{}{
			"patterns": []map[string]string{
				{"urlPattern": "*"},
			},
		},
	})
	resp := readRPC(t, ctx, conn)
	assertNoError(t, resp, 1)

	// Disable.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Fetch.disable",
	})
	resp = readRPC(t, ctx, conn)
	assertNoError(t, resp, 2)
}

func TestFetchRequestPausedAndContinue(t *testing.T) {
	const body = `<html><body>intercepted</body></html>`
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable Fetch interception.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Fetch.enable",
		"params": map[string]interface{}{
			"patterns": []map[string]string{
				{"urlPattern": "*"},
			},
		},
	})
	readRPC(t, ctx, conn) // enable response

	// Start navigation in background (it will be paused by interceptor).
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// Should receive Fetch.requestPaused event.
	var requestPaused map[string]interface{}
	for i := 0; i < 10; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if method, _ := msg["method"].(string); method == "Fetch.requestPaused" {
			requestPaused = msg
			break
		}
	}
	if requestPaused == nil {
		t.Fatal("did not receive Fetch.requestPaused event")
	}

	params := parseParams(t, requestPaused)
	reqID, ok := params["requestId"].(string)
	if !ok || reqID == "" {
		t.Fatal("requestPaused: missing requestId")
	}

	reqObj, ok := params["request"].(map[string]interface{})
	if !ok {
		t.Fatal("requestPaused: missing request object")
	}
	if reqObj["url"] != html.URL {
		t.Errorf("request url = %v, want %v", reqObj["url"], html.URL)
	}
	if reqObj["method"] != "GET" {
		t.Errorf("request method = %v, want GET", reqObj["method"])
	}

	// Continue the request.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     3,
		"method": "Fetch.continueRequest",
		"params": map[string]string{"requestId": reqID},
	})

	// Collect remaining messages: continueRequest response + navigate response + lifecycle events.
	var navigateOK bool
	for i := 0; i < 20; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			// Navigate response.
			if msg["error"] != nil {
				t.Errorf("navigate error: %v", msg["error"])
			}
			navigateOK = true
			break
		}
	}
	if !navigateOK {
		t.Error("did not receive successful Page.navigate response")
	}
}

func TestFetchFailRequest(t *testing.T) {
	var requestReceived bool
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable Fetch.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Fetch.enable",
	})
	readRPC(t, ctx, conn)

	// Navigate (will be intercepted).
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// Wait for requestPaused.
	var reqID string
	for i := 0; i < 10; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if method, _ := msg["method"].(string); method == "Fetch.requestPaused" {
			params := parseParams(t, msg)
			reqID = params["requestId"].(string)
			break
		}
	}
	if reqID == "" {
		t.Fatal("did not receive Fetch.requestPaused")
	}

	// Fail the request.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     3,
		"method": "Fetch.failRequest",
		"params": map[string]interface{}{
			"requestId": reqID,
			"reason":    "BlockedByClient",
		},
	})

	// Collect: failRequest response + navigate response (with errorText in result).
	var navigateErrored bool
	for i := 0; i < 10; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			// Page.navigate returns errorText in the result, not as an RPC error.
			if result, ok := msg["result"].(map[string]interface{}); ok {
				if et, ok := result["errorText"].(string); ok && et != "" {
					navigateErrored = true
				}
			}
			break
		}
	}
	if !navigateErrored {
		t.Error("expected navigate to return errorText after Fetch.failRequest")
	}
	if requestReceived {
		t.Error("HTTP request should not have been made after failRequest")
	}
}

func TestFetchPatternMatching(t *testing.T) {
	var hitCount atomic.Int32
	s, html := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only count the main document request, not background resource hints.
		if r.URL.Path == "/" || r.URL.Path == "" {
			hitCount.Add(1)
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Enable Fetch with a pattern that does NOT match the test server URL.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Fetch.enable",
		"params": map[string]interface{}{
			"patterns": []map[string]string{
				{"urlPattern": "https://never-match.example.com/*"},
			},
		},
	})
	readRPC(t, ctx, conn)

	// Navigate — should NOT be intercepted (no matching pattern).
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// Should get navigate response without any Fetch.requestPaused.
	var gotPaused bool
	for i := 0; i < 15; i++ {
		data := readRPC(t, ctx, conn)
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)
		if method, _ := msg["method"].(string); method == "Fetch.requestPaused" {
			gotPaused = true
		}
		if id, ok := msg["id"]; ok && int64(id.(float64)) == 2 {
			break
		}
	}
	if gotPaused {
		t.Error("Fetch.requestPaused should NOT fire for non-matching URL pattern")
	}
	if got := hitCount.Load(); got != 1 {
		t.Errorf("expected exactly 1 HTTP request, got %d", got)
	}
}

func TestFetchContinueInvalidRequestID(t *testing.T) {
	s, _ := startFetchServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html></html>"))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Fetch.enable",
	})
	readRPC(t, ctx, conn)

	// Try to continue a non-existent request.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Fetch.continueRequest",
		"params": map[string]string{"requestId": "does-not-exist"},
	})
	data := readRPC(t, ctx, conn)
	var resp struct {
		ID    int64 `json:"id"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(data, &resp)
	if resp.Error == nil {
		t.Error("expected error for invalid requestId")
	}
}

// --- helpers ---

func assertNoError(t *testing.T, data []byte, wantID int64) {
	t.Helper()
	var resp struct {
		ID    int64 `json:"id"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != wantID {
		t.Errorf("ID = %d, want %d", resp.ID, wantID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %s", resp.Error.Message)
	}
}
