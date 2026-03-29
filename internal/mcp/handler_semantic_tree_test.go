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

func TestSemanticTree_SPADynamicContent(t *testing.T) {
	// SPA scenario: initial HTML is a shell, JS dynamically builds the content
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<div id="app">Loading...</div>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)
	RegisterEvaluateTool(s)

	// Step 1: semantic_tree on initial (static) page — should have no meaningful structure
	args := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, result := callTool(s, "semantic_tree", args)
	if result == nil {
		t.Fatal("expected result")
	}
	initialText := result.Content[0].Text

	// Step 2: Execute JS to simulate SPA rendering — add nav, form, links
	jsScript := `
		var app = document.getElementById('app');
		app.textContent = '';

		var nav = document.createElement('nav');
		var a1 = document.createElement('a');
		a1.setAttribute('href', '/home');
		a1.textContent = 'Home';
		nav.appendChild(a1);
		var a2 = document.createElement('a');
		a2.setAttribute('href', '/about');
		a2.textContent = 'About';
		nav.appendChild(a2);
		app.appendChild(nav);

		var main = document.createElement('main');
		var h1 = document.createElement('h1');
		h1.textContent = 'Dashboard';
		main.appendChild(h1);

		var form = document.createElement('form');
		var input = document.createElement('input');
		input.setAttribute('type', 'text');
		input.setAttribute('placeholder', 'Search');
		form.appendChild(input);
		var btn = document.createElement('button');
		btn.setAttribute('type', 'submit');
		btn.textContent = 'Go';
		form.appendChild(btn);
		main.appendChild(form);

		app.appendChild(main);
		'rendered'
	`
	evalArgs := fmt.Sprintf(`{"url":%q,"script":%q}`, ts.URL, jsScript)
	_, evalResult := callTool(s, "evaluate", evalArgs)
	if evalResult == nil || evalResult.IsError {
		t.Fatalf("JS eval failed: %v", evalResult)
	}
	if evalResult.Content[0].Text != "rendered" {
		t.Fatalf("unexpected eval result: %s", evalResult.Content[0].Text)
	}

	// Step 3: semantic_tree again — should now reflect JS-rendered DOM
	_, result2 := callTool(s, "semantic_tree", args)
	if result2 == nil {
		t.Fatal("expected result after JS render")
	}
	if result2.IsError {
		t.Fatalf("unexpected error: %s", result2.Content[0].Text)
	}

	spaText := result2.Content[0].Text

	// Verify SPA-rendered content is visible in semantic tree
	if !strings.Contains(spaText, "[navigation]") {
		t.Errorf("SPA tree should contain [navigation], got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "[link#") {
		t.Errorf("SPA tree should contain links with NodeID, got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "Home") {
		t.Errorf("SPA tree should contain 'Home' link, got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "[main]") {
		t.Errorf("SPA tree should contain [main], got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "[heading]") {
		t.Errorf("SPA tree should contain [heading] for h1, got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "Dashboard") {
		t.Errorf("SPA tree should contain 'Dashboard' heading, got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "[textbox#") {
		t.Errorf("SPA tree should contain textbox for search input, got:\n%s", spaText)
	}
	if !strings.Contains(spaText, "[button#") {
		t.Errorf("SPA tree should contain submit button, got:\n%s", spaText)
	}

	// The SPA tree should have more content than the initial static tree
	if len(spaText) <= len(initialText) {
		t.Errorf("SPA tree (%d chars) should be richer than initial tree (%d chars)",
			len(spaText), len(initialText))
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
