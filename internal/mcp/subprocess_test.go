package mcp_test

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSubprocess_StartAndShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	p := startMCP(t)
	// Just starting and letting cleanup close stdin should work gracefully.
	_ = p
}

func TestSubprocess_InitializeHandshake(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	p := startMCP(t)
	p.initialize(t)
}

func TestSubprocess_ToolsList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	p := startMCP(t)
	p.initialize(t)

	resp, err := p.sendRequest("tools/list", nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("tools/list error: %v", resp.Error)
	}

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expectedTools := map[string]bool{
		"browse":        false,
		"evaluate":      false,
		"query":         false,
		"interact":      false,
		"semantic_tree": false,
	}
	for _, tool := range result.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}
	for name, found := range expectedTools {
		if !found {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestSubprocess_CustomTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	// Verify that custom timeout works (5 seconds should be plenty for handshake).
	p := startMCPWithTimeout(t, 5*time.Second)
	p.initialize(t)
}

func TestSubprocess_MethodNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	p := startMCP(t)
	p.initialize(t)

	resp, err := p.sendRequest("nonexistent/method", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown method, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601 (MethodNotFound)", resp.Error.Code)
	}
}

func TestSubprocess_CallToolHelper(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	p := startMCP(t)
	p.initialize(t)

	// Use callTool with browse on an invalid URL to verify the helper works.
	// We expect the tool to return a result (possibly with isError=true).
	result := p.callTool(t, "browse", map[string]any{
		"url": "http://127.0.0.1:1/nonexistent",
	})
	// The browse tool should return some content (error message about connection refused).
	if len(result.Content) == 0 {
		t.Error("expected at least one content item from browse tool")
	}
	// Verify Text() helper returns non-empty.
	text := result.Text()
	if text == "" {
		t.Error("expected non-empty text from result")
	}
}
