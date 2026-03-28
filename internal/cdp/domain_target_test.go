package cdp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/nyasuto/uzura/internal/cdp"
	"github.com/nyasuto/uzura/internal/page"
)

// readResponseByID reads messages until it finds a response with the given ID.
// Non-matching messages (events, other responses) are discarded.
func readResponseByID(t *testing.T, ctx context.Context, conn *websocket.Conn, id int64) json.RawMessage {
	t.Helper()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("readResponseByID(%d): %v", id, err)
		}
		var msg struct {
			ID int64 `json:"id"`
		}
		_ = json.Unmarshal(data, &msg)
		if msg.ID == id {
			return data
		}
	}
}

func setupTargetServer(t *testing.T) (*cdp.Server, *cdp.TargetDomain) {
	t.Helper()
	s := cdp.NewServer(cdp.WithAddr(":0"))
	td := cdp.NewTargetDomain(s, func() (*page.Page, error) {
		p := page.New(nil)
		return p, nil
	})
	td.Register(s)
	registerTestStubs(s)

	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	return s, td
}

func registerTestStubs(s *cdp.Server) {
	empty := func(_ json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(struct{}{})
	}
	s.Handle("Page.enable", empty)
	s.Handle("DOM.enable", empty)
	s.Handle("Runtime.enable", empty)
}

func TestTargetCreateTarget(t *testing.T) {
	s, td := setupTargetServer(t)

	// Add initial page.
	initial := page.New(nil)
	td.AddPage(initial, "default")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Create a new target with URL.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Target.createTarget",
		"params": map[string]interface{}{
			"url": "about:blank",
		},
	})

	respData := readResponseByID(t, ctx, conn, 1)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			TargetID string `json:"targetId"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Result.TargetID == "" {
		t.Fatal("expected targetId in response")
	}

	// Should have 2 targets now.
	targets := td.Targets()
	if len(targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(targets))
	}
}

func TestTargetCloseTarget(t *testing.T) {
	s, td := setupTargetServer(t)

	p := page.New(nil)
	td.AddPage(p, "default")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Target.closeTarget",
		"params": map[string]interface{}{
			"targetId": p.ID(),
		},
	})

	respData := readResponseByID(t, ctx, conn, 1)
	var resp struct {
		ID     int64 `json:"id"`
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Result.Success {
		t.Error("expected success=true")
	}

	targets := td.Targets()
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestTargetAttachDetach(t *testing.T) {
	s, td := setupTargetServer(t)

	p := page.New(nil)
	td.AddPage(p, "default")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Attach to target.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Target.attachToTarget",
		"params": map[string]interface{}{
			"targetId": p.ID(),
			"flatten":  true,
		},
	})

	respData := readRPC(t, ctx, conn)
	var attachResp struct {
		ID     int64 `json:"id"`
		Result struct {
			SessionID string `json:"sessionId"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respData, &attachResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	sessionID := attachResp.Result.SessionID
	if sessionID == "" {
		t.Fatal("expected sessionId in attach response")
	}

	// Detach from target.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Target.detachFromTarget",
		"params": map[string]interface{}{
			"sessionId": sessionID,
		},
	})

	respData = readRPC(t, ctx, conn)
	var detachResp struct {
		ID    int64           `json:"id"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(respData, &detachResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detachResp.ID != 2 {
		t.Errorf("response ID = %d, want 2", detachResp.ID)
	}
	if detachResp.Error != nil {
		t.Errorf("unexpected error: %s", detachResp.Error)
	}
}

func TestTargetSessionMultiplexing(t *testing.T) {
	s, td := setupTargetServer(t)

	// Register a simple per-page handler that returns the page URL.
	s.HandleSession("Page.getFrameTree", func(sess *cdp.Session, _ json.RawMessage) (json.RawMessage, []cdp.Event, error) {
		// This is the global handler - should not be reached for session-scoped requests.
		r, _ := json.Marshal(map[string]interface{}{
			"frameTree": map[string]interface{}{
				"frame": map[string]interface{}{
					"id":  "main",
					"url": "global-handler",
				},
			},
		})
		return r, nil, nil
	})

	// Create two pages.
	p1 := page.New(nil)
	td.AddPage(p1, "default")
	p2 := page.New(nil)
	td.AddPage(p2, "default")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Attach to page 1.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Target.attachToTarget",
		"params": map[string]interface{}{
			"targetId": p1.ID(),
			"flatten":  true,
		},
	})
	respData := readRPC(t, ctx, conn)
	var attach1 struct {
		Result struct {
			SessionID string `json:"sessionId"`
		} `json:"result"`
	}
	_ = json.Unmarshal(respData, &attach1)
	session1 := attach1.Result.SessionID

	// Attach to page 2.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Target.attachToTarget",
		"params": map[string]interface{}{
			"targetId": p2.ID(),
			"flatten":  true,
		},
	})
	respData = readRPC(t, ctx, conn)
	var attach2 struct {
		Result struct {
			SessionID string `json:"sessionId"`
		} `json:"result"`
	}
	_ = json.Unmarshal(respData, &attach2)
	session2 := attach2.Result.SessionID

	if session1 == session2 {
		t.Fatal("sessions must be different")
	}

	// Send request to page 1 via session1.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":        3,
		"method":    "Runtime.evaluate",
		"sessionId": session1,
		"params": map[string]interface{}{
			"expression": "1+1",
		},
	})
	resp1 := readRPC(t, ctx, conn)
	var r1 struct {
		ID        int64  `json:"id"`
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(resp1, &r1)
	if r1.ID != 3 {
		t.Errorf("response ID = %d, want 3", r1.ID)
	}
	if r1.SessionID != session1 {
		t.Errorf("sessionId = %q, want %q", r1.SessionID, session1)
	}

	// Send request to page 2 via session2.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":        4,
		"method":    "Runtime.evaluate",
		"sessionId": session2,
		"params": map[string]interface{}{
			"expression": "2+2",
		},
	})
	resp2 := readRPC(t, ctx, conn)
	var r2 struct {
		ID        int64  `json:"id"`
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(resp2, &r2)
	if r2.ID != 4 {
		t.Errorf("response ID = %d, want 4", r2.ID)
	}
	if r2.SessionID != session2 {
		t.Errorf("sessionId = %q, want %q", r2.SessionID, session2)
	}
}

func TestTargetDetachInvalidSession(t *testing.T) {
	s, _ := setupTargetServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Target.detachFromTarget",
		"params": map[string]interface{}{
			"sessionId": "nonexistent-session",
		},
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
		t.Fatal("expected error for invalid session")
	}
}

func TestTargetSessionMultiplexRouting(t *testing.T) {
	// Test that after detach, requests with that sessionId fail.
	s, td := setupTargetServer(t)

	p := page.New(nil)
	td.AddPage(p, "default")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := dialWS(t, ctx, s.Addr())
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Attach.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     1,
		"method": "Target.attachToTarget",
		"params": map[string]interface{}{
			"targetId": p.ID(),
			"flatten":  true,
		},
	})
	respData := readRPC(t, ctx, conn)
	var attach struct {
		Result struct {
			SessionID string `json:"sessionId"`
		} `json:"result"`
	}
	_ = json.Unmarshal(respData, &attach)
	sessionID := attach.Result.SessionID

	// Detach.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":     2,
		"method": "Target.detachFromTarget",
		"params": map[string]interface{}{
			"sessionId": sessionID,
		},
	})
	_ = readRPC(t, ctx, conn) // consume detach response

	// Send request with detached session — should get error.
	sendRPC(t, ctx, conn, map[string]interface{}{
		"id":        3,
		"method":    "Runtime.evaluate",
		"sessionId": sessionID,
		"params": map[string]interface{}{
			"expression": "1",
		},
	})

	respData = readRPC(t, ctx, conn)
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
		t.Fatal("expected error for detached session")
	}
}
