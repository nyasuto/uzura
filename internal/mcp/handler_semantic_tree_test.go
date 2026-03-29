package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSemanticTree_BasicPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<header>Site</header>
			<nav><a href="/">Home</a><a href="/about">About</a></nav>
			<main>
				<h1>Welcome</h1>
				<a href="/login">Login</a>
			</main>
			<footer>© 2026</footer>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)

	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "semantic_tree", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text

	// Should contain landmarks
	if !strings.Contains(text, "[banner]") {
		t.Errorf("output should contain [banner], got:\n%s", text)
	}
	if !strings.Contains(text, "[navigation]") {
		t.Errorf("output should contain [navigation], got:\n%s", text)
	}
	if !strings.Contains(text, "[main]") {
		t.Errorf("output should contain [main], got:\n%s", text)
	}
	if !strings.Contains(text, "[contentinfo]") {
		t.Errorf("output should contain [contentinfo], got:\n%s", text)
	}

	// Should contain interactive elements with NodeIDs
	if !strings.Contains(text, "[link#") {
		t.Errorf("output should contain link with NodeID, got:\n%s", text)
	}
	if !strings.Contains(text, "→ /") {
		t.Errorf("output should contain link value, got:\n%s", text)
	}

	// Should contain heading
	if !strings.Contains(text, "[heading]") {
		t.Errorf("output should contain heading, got:\n%s", text)
	}
}

func TestSemanticTree_FormPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<main>
				<h1>Login</h1>
				<form>
					<label for="email">Email</label>
					<input type="email" id="email">
					<label for="pass">Password</label>
					<input type="password" id="pass">
					<button type="submit">Sign In</button>
				</form>
			</main>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)

	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "semantic_tree", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text

	if !strings.Contains(text, "[textbox#") {
		t.Errorf("output should contain textbox, got:\n%s", text)
	}
	if !strings.Contains(text, "[button#") {
		t.Errorf("output should contain button, got:\n%s", text)
	}
	if !strings.Contains(text, "Email") {
		t.Errorf("output should contain 'Email' label, got:\n%s", text)
	}
}

func TestSemanticTree_EmptyPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><div>Just text</div></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)

	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "semantic_tree", args)

	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text
	if text != "(no semantic structure found)" {
		t.Errorf("expected empty message, got:\n%s", text)
	}
}

func TestSemanticTree_MissingURL(t *testing.T) {
	s := NewServer()
	RegisterSemanticTreeTool(s)

	resp, _ := callTool(s, "semantic_tree", `{}`)

	if resp.Error == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestSemanticTree_MaxDepth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<main>
				<nav><a href="/">Deep Link</a></nav>
			</main>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)

	// With max_depth=1, only top-level landmarks should appear
	args := fmt.Sprintf(`{"url":%q,"max_depth":1}`, ts.URL)
	_, result := callTool(s, "semantic_tree", args)

	if result == nil {
		t.Fatal("expected result")
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "[main]") {
		t.Errorf("should contain [main], got:\n%s", text)
	}
	// At depth 1, children should be truncated
	if strings.Contains(text, "[link#") {
		t.Errorf("at depth 1, links should be truncated, got:\n%s", text)
	}
}
