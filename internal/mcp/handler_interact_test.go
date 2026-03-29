package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInteract_Click(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<button id="btn">Click me</button>
			<script>
				var clicked = false;
				document.getElementById("btn").addEventListener("click", function() {
					clicked = true;
				});
			</script>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterInteractTool(s)

	args := fmt.Sprintf(`{"url":%q,"selector":"#btn","action":"click"}`, ts.URL)
	_, result := callTool(s, "interact", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	if result.Content[0].Text != "clicked" {
		t.Errorf("result = %q, want %q", result.Content[0].Text, "clicked")
	}
}

func TestInteract_Fill(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<input id="name" type="text" />
			<script>
				var lastInput = "";
				document.getElementById("name").addEventListener("input", function(e) {
					lastInput = e.target.value;
				});
			</script>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterInteractTool(s)

	args := fmt.Sprintf(`{"url":%q,"selector":"#name","action":"fill","value":"hello world"}`, ts.URL)
	_, result := callTool(s, "interact", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}
	if result.Content[0].Text != "filled" {
		t.Errorf("result = %q, want %q", result.Content[0].Text, "filled")
	}
}

func TestInteract_ElementNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><p>No button</p></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterInteractTool(s)

	args := fmt.Sprintf(`{"url":%q,"selector":"#nonexistent","action":"click"}`, ts.URL)
	_, result := callTool(s, "interact", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if !result.IsError {
		t.Error("expected isError=true for missing element")
	}
	if !strings.Contains(result.Content[0].Text, "no element matches") {
		t.Errorf("expected 'no element matches' message, got: %s", result.Content[0].Text)
	}
}

func TestInteract_InvalidAction(t *testing.T) {
	s := NewServer()
	RegisterInteractTool(s)

	args := `{"url":"http://example.com","selector":"p","action":"hover"}`
	resp, _ := callTool(s, "interact", args)

	if resp.Error == nil {
		t.Fatal("expected RPC error for invalid action")
	}
	if resp.Error.Code != CodeInvalidParams {
		t.Errorf("error code = %d, want %d", resp.Error.Code, CodeInvalidParams)
	}
}

func TestInteract_MissingParams(t *testing.T) {
	s := NewServer()
	RegisterInteractTool(s)

	tests := []struct {
		name string
		args string
	}{
		{"missing url", `{"selector":"p","action":"click"}`},
		{"missing selector", `{"url":"http://example.com","action":"click"}`},
		{"missing action", `{"url":"http://example.com","selector":"p"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := callTool(s, "interact", tt.args)
			if resp.Error == nil {
				t.Fatal("expected RPC error")
			}
		})
	}
}
