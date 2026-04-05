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

func TestBrowse_MarkdownFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
<title>Article Title</title>
<meta name="author" content="Jane Doe">
<meta name="description" content="Test article">
</head>
<body>
<nav>Menu</nav>
<article>
<h1>Article Title</h1>
<p>This is a substantial article paragraph with enough text for readability.
It should be extracted and converted to markdown format properly.</p>
<p>Second paragraph with <strong>bold</strong> and <a href="/link">a link</a>.</p>
<p>Third paragraph ensures sufficient content density for the readability
algorithm to confidently identify this as the main content area.</p>
</article>
<footer>Footer content</footer>
</body>
</html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q,"format":"markdown"}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	md := result.Content[0].Text

	// Should have frontmatter
	if !strings.Contains(md, "---\n") {
		t.Error("markdown output should contain frontmatter")
	}
	if !strings.Contains(md, "title:") {
		t.Error("markdown output should contain title in frontmatter")
	}

	// Should contain article content in markdown
	if !strings.Contains(md, "Article Title") {
		t.Errorf("markdown should contain article title: %s", md)
	}

	// Should not contain raw HTML tags
	if strings.Contains(md, "<article>") {
		t.Error("markdown output should not contain HTML tags")
	}
	if strings.Contains(md, "<nav>") {
		t.Error("markdown output should not contain nav")
	}
}

func TestBrowse_MarkdownFallback(t *testing.T) {
	// A page that readability can't extract should fall back to full page conversion
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Simple</title></head><body><p>Short content</p></body></html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q,"format":"markdown"}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	md := result.Content[0].Text
	if !strings.Contains(md, "---\n") {
		t.Error("fallback markdown should still have frontmatter")
	}
	if !strings.Contains(md, "Short content") {
		t.Errorf("fallback markdown should contain page content: %s", md)
	}
}

func TestBrowse_TextSkipsScriptStyleHidden(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
<script>var x = 1; console.log('noise');</script>
<style>body { color: red; font-size: 14px; }</style>
</head>
<body>
<p>Visible paragraph</p>
<script>alert('more noise');</script>
<div hidden>Hidden content</div>
<span aria-hidden="true">Icon glyph</span>
<div style="display:none">Invisible div</div>
<p>Another visible paragraph</p>
</body>
</html>`)
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "browse", args)

	if result == nil || result.IsError {
		t.Fatal("expected successful result")
	}
	text := result.Content[0].Text

	// Should contain visible text
	if !strings.Contains(text, "Visible paragraph") {
		t.Error("output should contain 'Visible paragraph'")
	}
	if !strings.Contains(text, "Another visible paragraph") {
		t.Error("output should contain 'Another visible paragraph'")
	}

	// Should NOT contain script content
	if strings.Contains(text, "console.log") {
		t.Errorf("should not contain script content, got: %s", text)
	}
	if strings.Contains(text, "alert(") {
		t.Errorf("should not contain script content 'alert', got: %s", text)
	}

	// Should NOT contain style content
	if strings.Contains(text, "font-size") {
		t.Errorf("should not contain style content, got: %s", text)
	}

	// Should NOT contain hidden content
	if strings.Contains(text, "Hidden content") {
		t.Errorf("should not contain hidden content, got: %s", text)
	}
	if strings.Contains(text, "Icon glyph") {
		t.Errorf("should not contain aria-hidden content, got: %s", text)
	}
	if strings.Contains(text, "Invisible div") {
		t.Errorf("should not contain display:none content, got: %s", text)
	}
}

func TestBrowse_TextSizeReduction(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html><head>`)
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, "<script>var longVar%d = %s;</script>", i, strings.Repeat("x", 200))
	}
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "<style>.c%d { bg: url(%s); }</style>", i, strings.Repeat("data", 100))
	}
	sb.WriteString(`</head><body><p>Real content here</p></body></html>`)
	rawHTML := sb.String()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(rawHTML))
	}))
	defer ts.Close()

	s := newTestServer(ts)
	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "browse", args)

	text := result.Content[0].Text
	if !strings.Contains(text, "Real content") {
		t.Error("output should contain real content")
	}

	ratio := float64(len(text)) / float64(len(rawHTML))
	t.Logf("Size reduction: raw=%d, output=%d, ratio=%.4f", len(rawHTML), len(text), ratio)
	if ratio > 0.1 {
		t.Errorf("output ratio = %.2f, expected < 0.1 (90%% reduction)", ratio)
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
