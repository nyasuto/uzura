package mcp

import (
	"encoding/json"
	"testing"
)

func TestServer_Initialize(t *testing.T) {
	s := NewServer()

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}}`
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

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ProtocolVersion != protocolVersion {
		t.Errorf("protocol version = %q, want %q", result.ProtocolVersion, protocolVersion)
	}
	if result.ServerInfo.Name != serverName {
		t.Errorf("server name = %q, want %q", result.ServerInfo.Name, serverName)
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability")
	}
}

func TestServer_Initialized(t *testing.T) {
	s := NewServer()

	// Send initialized notification (no id).
	notif := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	resp := s.HandleMessage([]byte(notif))
	if resp != nil {
		t.Errorf("notification should not return response, got: %s", resp)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.initialized {
		t.Error("server should be marked as initialized")
	}
}

func TestServer_Ping(t *testing.T) {
	s := NewServer()

	req := `{"jsonrpc":"2.0","id":99,"method":"ping"}`
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

	// Ping should return an empty object.
	if string(resp.Result) != "{}" {
		t.Errorf("result = %s, want {}", resp.Result)
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	s := NewServer()

	req := `{"jsonrpc":"2.0","id":1,"method":"nonexistent"}`
	respData := s.HandleMessage([]byte(req))
	if respData == nil {
		t.Fatal("expected response, got nil")
	}

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("error code = %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
}

func TestServer_InvalidJSON(t *testing.T) {
	s := NewServer()

	respData := s.HandleMessage([]byte(`{broken`))
	if respData == nil {
		t.Fatal("expected error response, got nil")
	}

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != CodeParseError {
		t.Errorf("error code = %d, want %d", resp.Error.Code, CodeParseError)
	}
}

func TestServer_CustomHandler(t *testing.T) {
	s := NewServer()
	s.Handle("tools/list", func(_ json.RawMessage) (any, error) {
		return map[string]any{"tools": []any{}}, nil
	})

	req := `{"jsonrpc":"2.0","id":5,"method":"tools/list"}`
	respData := s.HandleMessage([]byte(req))

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("expected tools array")
	}
	if len(tools) != 0 {
		t.Errorf("tools count = %d, want 0", len(tools))
	}
}

func TestServer_InitializeHandshake(t *testing.T) {
	s := NewServer()

	// Step 1: initialize request
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"claude-code","version":"1.0"}}}`
	respData := s.HandleMessage([]byte(initReq))
	if respData == nil {
		t.Fatal("expected initialize response")
	}

	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %v", resp.Error)
	}

	// Step 2: initialized notification
	notif := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	notifResp := s.HandleMessage([]byte(notif))
	if notifResp != nil {
		t.Errorf("notification should not return response, got: %s", notifResp)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.initialized {
		t.Error("server should be initialized after handshake")
	}
}

func TestServer_ResponseIDPreserved(t *testing.T) {
	s := NewServer()

	tests := []struct {
		name string
		req  string
		id   string
	}{
		{"integer id", `{"jsonrpc":"2.0","id":42,"method":"ping"}`, "42"},
		{"string id", `{"jsonrpc":"2.0","id":"abc","method":"ping"}`, `"abc"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respData := s.HandleMessage([]byte(tt.req))
			var resp Response
			if err := json.Unmarshal(respData, &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if resp.ID == nil {
				t.Fatal("expected id in response")
			}
			if string(*resp.ID) != tt.id {
				t.Errorf("id = %s, want %s", string(*resp.ID), tt.id)
			}
		})
	}
}
