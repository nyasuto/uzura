package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(ts *httptest.Server) *Server {
	_ = ts // just to hold reference
	s := NewServer()
	RegisterBrowseTool(s)
	return s
}

func callTool(s *Server, name, args string) (*Response, *ToolCallResult) {
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, args)
	respData := s.HandleMessage([]byte(req))
	var resp Response
	if err := json.Unmarshal(respData, &resp); err != nil {
		panic("unmarshal response: " + err.Error())
	}
	if resp.Error != nil {
		return &resp, nil
	}
	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		panic("unmarshal result: " + err.Error())
	}
	return &resp, &result
}

func TestBrowse_TextFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Test Page</title></head><body><h1>Hello World</h1><p>Content here.</p></body></html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content count = %d, want 1", len(result.Content))
	}
	text := result.Content[0].Text
	if !strings.Contains(text, "Hello World") {
		t.Errorf("text output should contain 'Hello World', got: %s", text)
	}
	if !strings.Contains(text, "Content here.") {
		t.Errorf("text output should contain 'Content here.', got: %s", text)
	}
}

func TestBrowse_HTMLFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><div id="main">Test</div></body></html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q,"format":"html"}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	html := result.Content[0].Text
	if !strings.Contains(html, `<div id="main">`) {
		t.Errorf("html output should contain div, got: %s", html)
	}
}

func TestBrowse_JSONFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><p>Hi</p></body></html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q,"format":"json"}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var tree map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &tree); err != nil {
		t.Fatalf("json output should be valid JSON: %v", err)
	}
	if tree["nodeName"] != "HTML" {
		t.Errorf("root nodeName = %v, want HTML", tree["nodeName"])
	}
}

func TestBrowse_NetworkError(t *testing.T) {
	s := NewServer()
	RegisterBrowseTool(s)

	// Use an invalid URL that will fail to connect.
	args := `{"url":"http://127.0.0.1:1"}`
	_, result := callTool(s, "browse", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if !result.IsError {
		t.Error("expected isError=true for network failure")
	}
	if !strings.Contains(result.Content[0].Text, "error") {
		t.Errorf("expected error message, got: %s", result.Content[0].Text)
	}
}

func TestBrowse_MissingURL(t *testing.T) {
	s := NewServer()
	RegisterBrowseTool(s)

	args := `{"format":"text"}`
	resp, _ := callTool(s, "browse", args)

	if resp.Error == nil {
		t.Fatal("expected RPC error for missing url")
	}
	if resp.Error.Code != CodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, CodeInvalidParams)
	}
}

func TestBrowse_DefaultFormatIsText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>Default format test</body></html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	// Default format is text — should NOT contain HTML tags.
	text := result.Content[0].Text
	if strings.Contains(text, "<body>") {
		t.Errorf("text format should not contain HTML tags, got: %s", text)
	}
	if !strings.Contains(text, "Default format test") {
		t.Errorf("text output should contain 'Default format test', got: %s", text)
	}
}
