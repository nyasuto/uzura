package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuery_MultipleElements(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><ul><li>One</li><li>Two</li><li>Three</li></ul></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterQueryTool(s)

	args := fmt.Sprintf(`{"url":%q,"selector":"li"}`, ts.URL)
	_, result := callTool(s, "query", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var results []QueryResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}
	expected := []string{"One", "Two", "Three"}
	for i, r := range results {
		if r.Text != expected[i] {
			t.Errorf("results[%d].text = %q, want %q", i, r.Text, expected[i])
		}
	}
}

func TestQuery_AttributeExtraction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><a href="/page1">Link 1</a><a href="/page2">Link 2</a></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterQueryTool(s)

	args := fmt.Sprintf(`{"url":%q,"selector":"a","attribute":"href"}`, ts.URL)
	_, result := callTool(s, "query", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	var results []QueryResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results count = %d, want 2", len(results))
	}
	if results[0].Value != "/page1" {
		t.Errorf("results[0].value = %q, want %q", results[0].Value, "/page1")
	}
	if results[1].Value != "/page2" {
		t.Errorf("results[1].value = %q, want %q", results[1].Value, "/page2")
	}
}

func TestQuery_NoMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><p>No tables here</p></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterQueryTool(s)

	args := fmt.Sprintf(`{"url":%q,"selector":"table"}`, ts.URL)
	_, result := callTool(s, "query", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatal("no match should not be an error")
	}

	var results []QueryResult
	if err := json.Unmarshal([]byte(result.Content[0].Text), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results count = %d, want 0", len(results))
	}
}

func TestQuery_MissingParams(t *testing.T) {
	s := NewServer()
	RegisterQueryTool(s)

	tests := []struct {
		name string
		args string
	}{
		{"missing url", `{"selector":"p"}`},
		{"missing selector", `{"url":"http://example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := callTool(s, "query", tt.args)
			if resp.Error == nil {
				t.Fatal("expected RPC error")
			}
		})
	}
}
