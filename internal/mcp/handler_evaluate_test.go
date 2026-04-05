package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEvaluate_BasicScript(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Test</h1></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterEvaluateTool(s)

	args := fmt.Sprintf(`{"url":%q,"script":"1 + 2"}`, ts.URL)
	_, result := callTool(s, "evaluate", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	if result.Content[0].Text != "3" {
		t.Errorf("result = %q, want %q", result.Content[0].Text, "3")
	}
}

func TestEvaluate_DOMAccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>My Page</title></head><body></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterEvaluateTool(s)

	args := fmt.Sprintf(`{"url":%q,"script":"document.title"}`, ts.URL)
	_, result := callTool(s, "evaluate", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	if result.Content[0].Text != "My Page" {
		t.Errorf("result = %q, want %q", result.Content[0].Text, "My Page")
	}
}

func TestEvaluate_ScriptError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterEvaluateTool(s)

	args := fmt.Sprintf(`{"url":%q,"script":"throw new Error('test error')"}`, ts.URL)
	_, result := callTool(s, "evaluate", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if !result.IsError {
		t.Error("expected isError=true for script error")
	}
	if !strings.Contains(result.Content[0].Text, "test error") {
		t.Errorf("error should contain 'test error', got: %s", result.Content[0].Text)
	}
}

func TestEvaluate_MissingParams(t *testing.T) {
	s := NewServer()
	RegisterEvaluateTool(s)

	tests := []struct {
		name string
		args string
	}{
		{"missing url", `{"script":"1+1"}`},
		{"missing script", `{"url":"http://example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := callTool(s, "evaluate", tt.args)
			if resp.Error == nil {
				t.Fatal("expected RPC error")
			}
			if resp.Error.Code != CodeInvalidParams {
				t.Errorf("error code = %d, want %d", resp.Error.Code, CodeInvalidParams)
			}
		})
	}
}

func TestEvaluate_MarkdownFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// SPA-like page with minimal initial content
		fmt.Fprint(w, `<html><head><title>SPA App</title></head><body><div id="root"></div></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterEvaluateTool(s)

	// Execute JS that adds content, then request markdown format
	script := `document.getElementById('root').innerHTML = '<h1>Hello from JS</h1><p>Content loaded via JavaScript.</p>'`
	args := fmt.Sprintf(`{"url":%q,"script":%q,"format":"markdown"}`, ts.URL, script)
	_, result := callTool(s, "evaluate", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	output := result.Content[0].Text
	if !strings.Contains(output, "Hello from JS") {
		t.Errorf("markdown should contain JS-generated content, got: %s", output)
	}
	if !strings.Contains(output, "title: SPA App") {
		t.Errorf("markdown should contain frontmatter, got: %s", output)
	}
}

func TestEvaluateToolDefinition(t *testing.T) {
	tool := EvaluateTool()

	if tool.Name != "evaluate" {
		t.Errorf("name = %q, want %q", tool.Name, "evaluate")
	}
	if tool.Description == "" {
		t.Error("description should not be empty")
	}
}
