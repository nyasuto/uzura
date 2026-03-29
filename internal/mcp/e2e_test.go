package mcp

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestE2E_InitializeToolsListShutdown(t *testing.T) {
	// Simulate a full MCP session: initialize → initialized → tools/list → EOF.
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	}, "\n") + "\n"

	var output strings.Builder
	var logOutput strings.Builder

	srv := NewServer()
	RegisterBrowseTool(srv)
	RegisterEvaluateTool(srv)
	RegisterQueryTool(srv)
	RegisterInteractTool(srv)

	tr := NewTransport(strings.NewReader(input), &output, &logOutput)
	err := srv.Serve(tr)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// Parse responses line by line.
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines (initialize + tools/list), got %d: %s", len(lines), output.String())
	}

	// Verify initialize response.
	var initResp Response
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("unmarshal init resp: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("init error: %v", initResp.Error)
	}

	// Verify tools/list response.
	var toolsResp Response
	if err := json.Unmarshal([]byte(lines[1]), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools resp: %v", err)
	}
	if toolsResp.Error != nil {
		t.Fatalf("tools/list error: %v", toolsResp.Error)
	}

	var result ToolsListResult
	if err := json.Unmarshal(toolsResp.Result, &result); err != nil {
		t.Fatalf("unmarshal tools result: %v", err)
	}
	if len(result.Tools) != 4 {
		t.Errorf("tools count = %d, want 4", len(result.Tools))
	}

	// Check tool names.
	names := make(map[string]bool)
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	for _, expected := range []string{"browse", "evaluate", "query", "interact"} {
		if !names[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestE2E_EmptyInput(t *testing.T) {
	var output strings.Builder
	srv := NewServer()
	tr := NewTransport(strings.NewReader(""), &output, io.Discard)
	err := srv.Serve(tr)
	if err != nil {
		t.Fatalf("Serve with empty input should return nil, got: %v", err)
	}
	if output.Len() != 0 {
		t.Errorf("expected no output, got: %s", output.String())
	}
}
