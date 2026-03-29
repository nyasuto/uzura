package mcp

import (
	"encoding/json"
	"testing"
)

func TestToolRegistry_Register(t *testing.T) {
	r := NewToolRegistry()

	tool := Tool{
		Name:        "browse",
		Description: "URLを開いてページのコンテンツを取得する",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`),
	}
	r.Register(tool)

	tools := r.List()
	if len(tools) != 1 {
		t.Fatalf("tools count = %d, want 1", len(tools))
	}
	if tools[0].Name != "browse" {
		t.Errorf("tool name = %q, want %q", tools[0].Name, "browse")
	}
}

func TestToolRegistry_DuplicateOverwrites(t *testing.T) {
	r := NewToolRegistry()

	r.Register(Tool{Name: "browse", Description: "v1"})
	r.Register(Tool{Name: "browse", Description: "v2"})

	tools := r.List()
	if len(tools) != 1 {
		t.Fatalf("tools count = %d, want 1", len(tools))
	}
	if tools[0].Description != "v2" {
		t.Errorf("description = %q, want %q", tools[0].Description, "v2")
	}
}

func TestToolsListHandler(t *testing.T) {
	s := NewServer()

	browse := Tool{
		Name:        "browse",
		Description: "URLを開いてページのコンテンツを取得する",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": { "type": "string", "description": "取得するURL" },
				"format": {
					"type": "string",
					"enum": ["text", "html", "json"],
					"default": "text"
				}
			},
			"required": ["url"]
		}`),
	}
	s.Tools.Register(browse)

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	respData := s.HandleMessage([]byte(req))
	if respData == nil {
		t.Fatal("expected response, got nil")
	}

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("tools count = %d, want 1", len(result.Tools))
	}

	tool := result.Tools[0]
	if tool.Name != "browse" {
		t.Errorf("name = %q, want %q", tool.Name, "browse")
	}
	if tool.Description != "URLを開いてページのコンテンツを取得する" {
		t.Errorf("description = %q", tool.Description)
	}

	// Verify inputSchema has required fields.
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
	props, hasProps := schema["properties"].(map[string]any)
	if !hasProps {
		t.Fatal("expected properties object")
	}
	if _, hasURL := props["url"]; !hasURL {
		t.Error("missing url property")
	}
	if _, hasFmt := props["format"]; !hasFmt {
		t.Error("missing format property")
	}
	req2, hasReq := schema["required"].([]any)
	if !hasReq {
		t.Fatal("expected required array")
	}
	if len(req2) != 1 || req2[0] != "url" {
		t.Errorf("required = %v, want [url]", req2)
	}
}

func TestToolsListEmpty(t *testing.T) {
	s := NewServer()

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	respData := s.HandleMessage([]byte(req))

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Tools) != 0 {
		t.Errorf("tools count = %d, want 0", len(result.Tools))
	}
}

func TestBrowseToolDefinition(t *testing.T) {
	tool := BrowseTool()

	if tool.Name != "browse" {
		t.Errorf("name = %q, want %q", tool.Name, "browse")
	}

	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	// Check url property.
	urlProp := props["url"].(map[string]any)
	if urlProp["type"] != "string" {
		t.Errorf("url type = %v, want string", urlProp["type"])
	}

	// Check format property.
	fmtProp := props["format"].(map[string]any)
	if fmtProp["type"] != "string" {
		t.Errorf("format type = %v, want string", fmtProp["type"])
	}
	enumVals := fmtProp["enum"].([]any)
	expected := []string{"text", "html", "json"}
	if len(enumVals) != len(expected) {
		t.Fatalf("enum count = %d, want %d", len(enumVals), len(expected))
	}
	for i, v := range enumVals {
		if v != expected[i] {
			t.Errorf("enum[%d] = %v, want %v", i, v, expected[i])
		}
	}
	if fmtProp["default"] != "text" {
		t.Errorf("format default = %v, want text", fmtProp["default"])
	}

	// Check required.
	reqArr := schema["required"].([]any)
	if len(reqArr) != 1 || reqArr[0] != "url" {
		t.Errorf("required = %v, want [url]", reqArr)
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	s := NewServer()

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}`
	respData := s.HandleMessage([]byte(req))

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != CodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, CodeInvalidParams)
	}
}
