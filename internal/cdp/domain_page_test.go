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

func startPageServer(t *testing.T) (srv *cdp.Server, htmlSrv *httptest.Server) {
	t.Helper()

	// HTML server for navigation targets.
	html := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Test Page</title></head><body><h1>Hello</h1></body></html>`))
	}))
	t.Cleanup(html.Close)

	fetcher := network.NewFetcher(nil)
	p := page.New(&page.Options{Fetcher: fetcher})

	s := cdp.NewServer(cdp.WithAddr(":0"))
	domain := cdp.NewPageDomain(p)
	domain.Register(s)

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

type eventMsg struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func readAllEvents(t *testing.T, ctx context.Context, conn *websocket.Conn, n int) []eventMsg {
	t.Helper()
	events := make([]eventMsg, n)
	for i := 0; i < n; i++ {
		data := readRPC(t, ctx, conn)
		if err := json.Unmarshal(data, &events[i]); err != nil {
			t.Fatalf("unmarshal event %d: %v", i, err)
		}
	}
	return events
}

func TestPageEnable(t *testing.T) {
	s, _ := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.enable",
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
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
}

func TestPageNavigate(t *testing.T) {
	s, html := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Navigate to the test HTML server.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// First message: the RPC response with frameId.
	respData := readRPC(t, ctx, conn)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			FrameID   string `json:"frameId"`
			ErrorText string `json:"errorText,omitempty"`
		} `json:"result"`
		Error *struct {
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
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.Result.FrameID != "main" {
		t.Errorf("frameId = %q, want %q", resp.Result.FrameID, "main")
	}
	if resp.Result.ErrorText != "" {
		t.Errorf("unexpected errorText: %q", resp.Result.ErrorText)
	}

	// Read all lifecycle events following the response.
	events := readAllEvents(t, ctx, conn, 8)

	// Verify key events are present.
	methodSet := make(map[string]json.RawMessage)
	for _, evt := range events {
		methodSet[evt.Method] = evt.Params
	}
	for _, want := range []string{"Page.domContentEventFired", "Page.loadEventFired", "Page.frameNavigated", "Page.lifecycleEvent"} {
		if _, ok := methodSet[want]; !ok {
			t.Errorf("missing expected event %q", want)
		}
	}

	// Verify domContentEventFired has a timestamp.
	if params, ok := methodSet["Page.domContentEventFired"]; ok {
		var ts struct {
			Timestamp float64 `json:"timestamp"`
		}
		json.Unmarshal(params, &ts)
		if ts.Timestamp <= 0 {
			t.Errorf("domContentEventFired timestamp = %f, want > 0", ts.Timestamp)
		}
	}
}

func TestPageNavigateMissingURL(t *testing.T) {
	s, _ := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]string{},
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
		t.Fatal("expected error for missing URL")
	}
}

func TestPageNavigateInvalidURL(t *testing.T) {
	s, _ := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]string{"url": "http://127.0.0.1:1/nonexistent"},
	})

	respData := readRPC(t, ctx, conn)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			FrameID   string `json:"frameId"`
			ErrorText string `json:"errorText"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Result.FrameID != "main" {
		t.Errorf("frameId = %q, want %q", resp.Result.FrameID, "main")
	}
	if resp.Result.ErrorText == "" {
		t.Error("expected errorText for invalid URL")
	}
}

func TestPageGetFrameTree(t *testing.T) {
	s, html := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Navigate first.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})
	// Consume response + 8 lifecycle events.
	for range 9 {
		readRPC(t, ctx, conn)
	}

	// Get frame tree.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.getFrameTree",
	})

	respData := readRPC(t, ctx, conn)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			FrameTree struct {
				Frame struct {
					ID       string `json:"id"`
					URL      string `json:"url"`
					MimeType string `json:"mimeType"`
				} `json:"frame"`
			} `json:"frameTree"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != 2 {
		t.Errorf("ID = %d, want 2", resp.ID)
	}
	if resp.Result.FrameTree.Frame.ID != "main" {
		t.Errorf("frame ID = %q, want %q", resp.Result.FrameTree.Frame.ID, "main")
	}
	if resp.Result.FrameTree.Frame.URL != html.URL {
		t.Errorf("frame URL = %q, want %q", resp.Result.FrameTree.Frame.URL, html.URL)
	}
	if resp.Result.FrameTree.Frame.MimeType != "text/html" {
		t.Errorf("mimeType = %q, want %q", resp.Result.FrameTree.Frame.MimeType, "text/html")
	}
}

func TestPageGetFrameTreeBeforeNavigate(t *testing.T) {
	s, _ := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.getFrameTree",
	})

	respData := readRPC(t, ctx, conn)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			FrameTree struct {
				Frame struct {
					URL string `json:"url"`
				} `json:"frame"`
			} `json:"frameTree"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Result.FrameTree.Frame.URL != "about:blank" {
		t.Errorf("URL = %q, want %q", resp.Result.FrameTree.Frame.URL, "about:blank")
	}
}

func TestPageNavigateFullFlow(t *testing.T) {
	s, html := startPageServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 1. Page.enable
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Page.enable",
	})
	enableResp := readRPC(t, ctx, conn)
	var er struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(enableResp, &er)
	if er.ID != 1 {
		t.Errorf("enable response ID = %d, want 1", er.ID)
	}

	// 2. Page.navigate
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Page.navigate",
		"params": map[string]string{"url": html.URL},
	})

	// Response
	navResp := readRPC(t, ctx, conn)
	var nr struct {
		ID     int64 `json:"id"`
		Result struct {
			FrameID string `json:"frameId"`
		} `json:"result"`
	}
	json.Unmarshal(navResp, &nr)
	if nr.ID != 2 {
		t.Errorf("navigate response ID = %d, want 2", nr.ID)
	}
	if nr.Result.FrameID != "main" {
		t.Errorf("frameId = %q, want main", nr.Result.FrameID)
	}

	// Read all lifecycle events and verify key events are present.
	events := readAllEvents(t, ctx, conn, 8)
	methodSet := make(map[string]bool)
	for _, evt := range events {
		methodSet[evt.Method] = true
	}
	if !methodSet["Page.domContentEventFired"] {
		t.Error("missing Page.domContentEventFired event")
	}
	if !methodSet["Page.loadEventFired"] {
		t.Error("missing Page.loadEventFired event")
	}
	if !methodSet["Page.frameNavigated"] {
		t.Error("missing Page.frameNavigated event")
	}

	// 3. Page.getFrameTree — verify navigation worked
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     3,
		"method": "Page.getFrameTree",
	})
	ftResp := readRPC(t, ctx, conn)
	var ft struct {
		Result struct {
			FrameTree struct {
				Frame struct {
					URL string `json:"url"`
				} `json:"frame"`
			} `json:"frameTree"`
		} `json:"result"`
	}
	json.Unmarshal(ftResp, &ft)
	if ft.Result.FrameTree.Frame.URL != html.URL {
		t.Errorf("frame URL = %q, want %q", ft.Result.FrameTree.Frame.URL, html.URL)
	}
}
