package mcp

import (
	"encoding/json"
	"testing"
)

func TestParseRequest_Valid(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		method string
		hasID  bool
	}{
		{
			name:   "request with params",
			input:  `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			method: "initialize",
			hasID:  true,
		},
		{
			name:   "request without params",
			input:  `{"jsonrpc":"2.0","id":42,"method":"ping"}`,
			method: "ping",
			hasID:  true,
		},
		{
			name:   "notification (no id)",
			input:  `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			method: "notifications/initialized",
			hasID:  false,
		},
		{
			name:   "string id",
			input:  `{"jsonrpc":"2.0","id":"abc","method":"ping"}`,
			method: "ping",
			hasID:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := ParseRequest([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Method != tt.method {
				t.Errorf("method = %q, want %q", req.Method, tt.method)
			}
			if tt.hasID && req.IsNotification() {
				t.Error("expected request, got notification")
			}
			if !tt.hasID && !req.IsNotification() {
				t.Error("expected notification, got request")
			}
		})
	}
}

func TestParseRequest_Errors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCode int
	}{
		{
			name:     "invalid JSON",
			input:    `{not json}`,
			wantCode: CodeParseError,
		},
		{
			name:     "wrong jsonrpc version",
			input:    `{"jsonrpc":"1.0","id":1,"method":"ping"}`,
			wantCode: CodeInvalidRequest,
		},
		{
			name:     "missing method",
			input:    `{"jsonrpc":"2.0","id":1}`,
			wantCode: CodeInvalidRequest,
		},
		{
			name:     "empty string",
			input:    ``,
			wantCode: CodeParseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRequest([]byte(tt.input))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			rpcErr, ok := err.(*RPCError)
			if !ok {
				t.Fatalf("expected *RPCError, got %T", err)
			}
			if rpcErr.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", rpcErr.Code, tt.wantCode)
			}
		})
	}
}

func TestResponseRoundTrip(t *testing.T) {
	id := json.RawMessage(`1`)
	resp, err := NewResponse(&id, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("NewResponse: %v", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Response
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q, want %q", got.JSONRPC, "2.0")
	}
	if got.Error != nil {
		t.Errorf("unexpected error: %v", got.Error)
	}

	var result map[string]string
	if err := json.Unmarshal(got.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("result[key] = %q, want %q", result["key"], "value")
	}
}

func TestErrorResponseRoundTrip(t *testing.T) {
	id := json.RawMessage(`2`)
	resp := NewErrorResponse(&id, &RPCError{
		Code:    CodeMethodNotFound,
		Message: "method not found: foo",
	})

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Response
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Error == nil {
		t.Fatal("expected error in response")
	}
	if got.Error.Code != CodeMethodNotFound {
		t.Errorf("error code = %d, want %d", got.Error.Code, CodeMethodNotFound)
	}
}

func TestNotificationRoundTrip(t *testing.T) {
	notif, err := NewNotification("notifications/initialized", nil)
	if err != nil {
		t.Fatalf("NewNotification: %v", err)
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want %q", got["jsonrpc"], "2.0")
	}
	if got["method"] != "notifications/initialized" {
		t.Errorf("method = %v, want %q", got["method"], "notifications/initialized")
	}
	if _, hasID := got["id"]; hasID {
		t.Error("notification should not have id")
	}
}
