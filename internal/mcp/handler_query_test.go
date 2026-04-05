package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func parseQueryResponse(t *testing.T, result *ToolCallResult) QueryResponse {
	t.Helper()
	var resp QueryResponse
	if err := json.Unmarshal([]byte(result.Content[0].Text), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

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

	resp := parseQueryResponse(t, result)
	if resp.Total != 3 {
		t.Fatalf("total = %d, want 3", resp.Total)
	}
	if resp.Returned != 3 {
		t.Fatalf("returned = %d, want 3", resp.Returned)
	}
	expected := []string{"One", "Two", "Three"}
	for i, r := range resp.Results {
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

	resp := parseQueryResponse(t, result)
	if resp.Returned != 2 {
		t.Fatalf("returned = %d, want 2", resp.Returned)
	}
	if resp.Results[0].Value != "/page1" {
		t.Errorf("results[0].value = %q, want %q", resp.Results[0].Value, "/page1")
	}
	if resp.Results[1].Value != "/page2" {
		t.Errorf("results[1].value = %q, want %q", resp.Results[1].Value, "/page2")
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

	resp := parseQueryResponse(t, result)
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if resp.Returned != 0 {
		t.Errorf("returned = %d, want 0", resp.Returned)
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

func TestQuery_LimitAndOffset(t *testing.T) {
	// Build a page with 200 list items
	var items strings.Builder
	for i := 1; i <= 200; i++ {
		fmt.Fprintf(&items, "<li>Item %d</li>", i)
	}
	html := fmt.Sprintf(`<html><body><ul>%s</ul></body></html>`, items.String())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterQueryTool(s)

	t.Run("default limit 100", func(t *testing.T) {
		args := fmt.Sprintf(`{"url":%q,"selector":"li"}`, ts.URL)
		_, result := callTool(s, "query", args)
		resp := parseQueryResponse(t, result)
		if resp.Total != 200 {
			t.Errorf("total = %d, want 200", resp.Total)
		}
		if resp.Returned != 100 {
			t.Errorf("returned = %d, want 100", resp.Returned)
		}
		if resp.Offset != 0 {
			t.Errorf("offset = %d, want 0", resp.Offset)
		}
		if resp.Results[0].Text != "Item 1" {
			t.Errorf("first = %q, want 'Item 1'", resp.Results[0].Text)
		}
	})

	t.Run("custom limit", func(t *testing.T) {
		args := fmt.Sprintf(`{"url":%q,"selector":"li","limit":10}`, ts.URL)
		_, result := callTool(s, "query", args)
		resp := parseQueryResponse(t, result)
		if resp.Total != 200 {
			t.Errorf("total = %d, want 200", resp.Total)
		}
		if resp.Returned != 10 {
			t.Errorf("returned = %d, want 10", resp.Returned)
		}
	})

	t.Run("offset pagination", func(t *testing.T) {
		args := fmt.Sprintf(`{"url":%q,"selector":"li","limit":5,"offset":195}`, ts.URL)
		_, result := callTool(s, "query", args)
		resp := parseQueryResponse(t, result)
		if resp.Total != 200 {
			t.Errorf("total = %d, want 200", resp.Total)
		}
		if resp.Returned != 5 {
			t.Errorf("returned = %d, want 5", resp.Returned)
		}
		if resp.Offset != 195 {
			t.Errorf("offset = %d, want 195", resp.Offset)
		}
		if resp.Results[0].Text != "Item 196" {
			t.Errorf("first = %q, want 'Item 196'", resp.Results[0].Text)
		}
	})

	t.Run("offset beyond total", func(t *testing.T) {
		args := fmt.Sprintf(`{"url":%q,"selector":"li","offset":999}`, ts.URL)
		_, result := callTool(s, "query", args)
		resp := parseQueryResponse(t, result)
		if resp.Total != 200 {
			t.Errorf("total = %d, want 200", resp.Total)
		}
		if resp.Returned != 0 {
			t.Errorf("returned = %d, want 0", resp.Returned)
		}
	})
}
