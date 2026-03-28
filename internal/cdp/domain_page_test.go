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

	// Next two messages should be events.
	events := make([]struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}, 2)

	for i := 0; i < 2; i++ {
		data := readRPC(t, ctx, conn)
		if err := json.Unmarshal(data, &events[i]); err != nil {
			t.Fatalf("unmarshal event %d: %v", i, err)
		}
	}

	if events[0].Method != "Page.domContentEventFired" {
		t.Errorf("event[0] = %q, want Page.domContentEventFired", events[0].Method)
	}
	if events[1].Method != "Page.loadEventFired" {
		t.Errorf("event[1] = %q, want Page.loadEventFired", events[1].Method)
	}

	// Verify timestamps exist in events.
	for i, evt := range events {
		var params struct {
			Timestamp float64 `json:"timestamp"`
		}
		if err := json.Unmarshal(evt.Params, &params); err != nil {
			t.Fatalf("event %d params: %v", i, err)
		}
		if params.Timestamp <= 0 {
			t.Errorf("event %d: timestamp = %f, want > 0", i, params.Timestamp)
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
	// Consume response + 2 events.
	readRPC(t, ctx, conn)
	readRPC(t, ctx, conn)
	readRPC(t, ctx, conn)

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

	// Events: domContentEventFired, then loadEventFired
	evt1 := readRPC(t, ctx, conn)
	evt2 := readRPC(t, ctx, conn)

	var e1, e2 struct {
		Method string `json:"method"`
	}
	json.Unmarshal(evt1, &e1)
	json.Unmarshal(evt2, &e2)

	if e1.Method != "Page.domContentEventFired" {
		t.Errorf("first event = %q, want Page.domContentEventFired", e1.Method)
	}
	if e2.Method != "Page.loadEventFired" {
		t.Errorf("second event = %q, want Page.loadEventFired", e2.Method)
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
