package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSemanticTreeThenInteract(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body>
			<main>
				<form>
					<label for="email">Email</label>
					<input type="email" id="email" placeholder="you@example.com">
					<button type="submit" id="btn">Login</button>
				</form>
				<script>
					document.getElementById("btn").addEventListener("click", function(e) {
						document.title = "clicked:" + document.getElementById("email").value;
					});
				</script>
			</main>
		</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)
	RegisterInteractTool(s)

	// Step 1: Get semantic tree to discover NodeIDs
	stArgs := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, stResult := callTool(s, "semantic_tree", stArgs)
	if stResult == nil {
		t.Fatal("expected semantic_tree result")
	}
	if stResult.IsError {
		t.Fatalf("semantic_tree error: %s", stResult.Content[0].Text)
	}

	treeOutput := stResult.Content[0].Text
	t.Logf("Semantic tree:\n%s", treeOutput)

	// Find the textbox NodeID (should be the email input)
	textboxID := extractNodeID(t, treeOutput, "textbox")
	buttonID := extractNodeID(t, treeOutput, "button")

	if textboxID == 0 {
		t.Fatal("could not find textbox NodeID in semantic tree output")
	}
	if buttonID == 0 {
		t.Fatal("could not find button NodeID in semantic tree output")
	}

	// Step 2: Fill the email field using node:N selector
	fillArgs := fmt.Sprintf(`{"url":%q,"selector":"node:%d","action":"fill","value":"test@example.com"}`, ts.URL, textboxID)
	_, fillResult := callTool(s, "interact", fillArgs)
	if fillResult == nil {
		t.Fatal("expected fill result")
	}
	if fillResult.IsError {
		t.Fatalf("fill error: %s", fillResult.Content[0].Text)
	}
	if fillResult.Content[0].Text != "filled" {
		t.Errorf("fill result = %q, want 'filled'", fillResult.Content[0].Text)
	}

	// Step 3: Click the button using node:N selector
	clickArgs := fmt.Sprintf(`{"url":%q,"selector":"node:%d","action":"click"}`, ts.URL, buttonID)
	_, clickResult := callTool(s, "interact", clickArgs)
	if clickResult == nil {
		t.Fatal("expected click result")
	}
	if clickResult.IsError {
		t.Fatalf("click error: %s", clickResult.Content[0].Text)
	}
	if clickResult.Content[0].Text != "clicked" {
		t.Errorf("click result = %q, want 'clicked'", clickResult.Content[0].Text)
	}
}

func TestInteractNodeSelectorWithoutSemanticTree(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><button>Click</button></body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterInteractTool(s)

	// Using node:1 without running semantic_tree first should fail gracefully
	args := fmt.Sprintf(`{"url":%q,"selector":"node:1","action":"click"}`, ts.URL)
	_, result := callTool(s, "interact", args)
	if result == nil {
		t.Fatal("expected result")
	}
	if !result.IsError {
		t.Error("expected error when using node selector without semantic_tree")
	}
	if !strings.Contains(result.Content[0].Text, "run semantic_tree first") {
		t.Errorf("error message should mention semantic_tree, got: %s", result.Content[0].Text)
	}
}

func TestParseNodeSelector(t *testing.T) {
	tests := []struct {
		input string
		id    int
		ok    bool
	}{
		{"node:1", 1, true},
		{"node:42", 42, true},
		{"node:0", 0, false},
		{"node:-1", 0, false},
		{"node:abc", 0, false},
		{"#btn", 0, false},
		{".class", 0, false},
		{"", 0, false},
	}

	for _, tt := range tests {
		id, ok := parseNodeSelector(tt.input)
		if ok != tt.ok || id != tt.id {
			t.Errorf("parseNodeSelector(%q) = (%d, %v), want (%d, %v)", tt.input, id, ok, tt.id, tt.ok)
		}
	}
}

func TestLoginWorkflow_SemanticTreeToInteract(t *testing.T) {
	// E2E workflow simulating an AI agent's "log in to this site" task:
	// 1. semantic_tree → understand page structure
	// 2. evaluate → register event handlers (simulating <script> auto-execution)
	// 3. interact (fill) → enter email and password
	// 4. interact (click) → submit the form
	// 5. evaluate → verify login result
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Login - MyApp</title></head>
<body>
<header><nav><a href="/">MyApp</a><a href="/about">About</a></nav></header>
<main>
  <h1>Sign In</h1>
  <form id="loginForm">
    <label for="email">Email Address</label>
    <input type="email" id="email" name="email" placeholder="you@example.com">
    <label for="password">Password</label>
    <input type="password" id="password" name="password" placeholder="Your password">
    <label for="remember"><input type="checkbox" id="remember" name="remember"> Remember me</label>
    <button type="submit" id="loginBtn">Sign In</button>
  </form>
  <p id="status">Not logged in</p>
</main>
<footer>© 2026 MyApp</footer>
</body></html>`)
	}))
	defer ts.Close()

	s := NewServer()
	RegisterSemanticTreeTool(s)
	RegisterInteractTool(s)
	RegisterEvaluateTool(s)

	// Step 1: Get semantic tree to understand the page structure
	stArgs := fmt.Sprintf(`{"url":%q}`, ts.URL)
	_, stResult := callTool(s, "semantic_tree", stArgs)
	if stResult == nil || stResult.IsError {
		t.Fatalf("semantic_tree failed: %v", stResult)
	}

	tree := stResult.Content[0].Text
	t.Logf("Semantic tree:\n%s", tree)

	// Verify the tree shows a login form with expected elements
	for _, expected := range []string{
		"[banner]", "[navigation]", "[main]", "[contentinfo]",
		"[heading]", "Sign In",
		"[textbox#", "Email",
		"[textbox#", "Password",
		"[checkbox#",
		"[button#", "Sign In",
	} {
		if !strings.Contains(tree, expected) {
			t.Errorf("semantic tree should contain %q, got:\n%s", expected, tree)
		}
	}

	// Step 2: Extract NodeIDs for email, password, and submit button
	emailID := extractNodeID(t, tree, "textbox")
	if emailID == 0 {
		t.Fatal("could not find email textbox NodeID")
	}

	firstTextboxEnd := strings.Index(tree, fmt.Sprintf("[textbox#%d]", emailID))
	remainingTree := tree[firstTextboxEnd+10:]
	passwordID := extractNodeID(t, remainingTree, "textbox")
	if passwordID == 0 {
		t.Fatal("could not find password textbox NodeID")
	}

	submitID := extractNodeID(t, tree, "button")
	if submitID == 0 {
		t.Fatal("could not find submit button NodeID")
	}

	t.Logf("Found NodeIDs — email: %d, password: %d, submit: %d", emailID, passwordID, submitID)

	// Step 3: Register click handler via evaluate (simulating page's JS)
	registerScript := `
		document.getElementById("loginBtn").addEventListener("click", function(e) {
			var email = document.getElementById("email").value;
			var pass = document.getElementById("password").value;
			if (email && pass) {
				document.getElementById("status").textContent = "Welcome, " + email + "!";
				document.title = "Dashboard - MyApp";
			} else {
				document.getElementById("status").textContent = "Please fill all fields";
			}
		});
		"handlers registered"
	`
	evalSetup := fmt.Sprintf(`{"url":%q,"script":%q}`, ts.URL, registerScript)
	_, setupResult := callTool(s, "evaluate", evalSetup)
	if setupResult == nil || setupResult.IsError {
		t.Fatalf("evaluate setup failed: %v", setupResult)
	}

	// Step 4: Fill email
	fillEmail := fmt.Sprintf(`{"url":%q,"selector":"node:%d","action":"fill","value":"user@myapp.com"}`,
		ts.URL, emailID)
	_, fillResult := callTool(s, "interact", fillEmail)
	if fillResult == nil || fillResult.IsError {
		t.Fatalf("fill email failed: %v", fillResult)
	}

	// Step 5: Fill password
	fillPass := fmt.Sprintf(`{"url":%q,"selector":"node:%d","action":"fill","value":"secret123"}`,
		ts.URL, passwordID)
	_, fillResult2 := callTool(s, "interact", fillPass)
	if fillResult2 == nil || fillResult2.IsError {
		t.Fatalf("fill password failed: %v", fillResult2)
	}

	// Step 6: Click submit button
	clickSubmit := fmt.Sprintf(`{"url":%q,"selector":"node:%d","action":"click"}`,
		ts.URL, submitID)
	_, clickResult := callTool(s, "interact", clickSubmit)
	if clickResult == nil || clickResult.IsError {
		t.Fatalf("click submit failed: %v", clickResult)
	}

	// Step 7: Verify login result via evaluate
	evalArgs := fmt.Sprintf(`{"url":%q,"script":"document.getElementById('status').textContent"}`, ts.URL)
	_, evalResult := callTool(s, "evaluate", evalArgs)
	if evalResult == nil || evalResult.IsError {
		t.Fatalf("evaluate failed: %v", evalResult)
	}
	statusText := evalResult.Content[0].Text
	if !strings.Contains(statusText, "Welcome, user@myapp.com!") {
		t.Errorf("expected welcome message, got: %s", statusText)
	}

	// Verify page title changed
	evalTitle := fmt.Sprintf(`{"url":%q,"script":"document.title"}`, ts.URL)
	_, titleResult := callTool(s, "evaluate", evalTitle)
	if titleResult == nil || titleResult.IsError {
		t.Fatalf("evaluate title failed: %v", titleResult)
	}
	if titleResult.Content[0].Text != "Dashboard - MyApp" {
		t.Errorf("expected title 'Dashboard - MyApp', got: %s", titleResult.Content[0].Text)
	}
}

// extractNodeID finds the first NodeID for a given role in formatted output.
// Format: [role#N] or [role#N] name
func extractNodeID(t *testing.T, output, role string) int {
	t.Helper()
	prefix := "[" + role + "#"
	idx := strings.Index(output, prefix)
	if idx < 0 {
		return 0
	}
	rest := output[idx+len(prefix):]
	endIdx := strings.Index(rest, "]")
	if endIdx < 0 {
		return 0
	}
	idStr := rest[:endIdx]
	var id int
	if err := json.Unmarshal([]byte(idStr), &id); err != nil {
		// Try direct parse
		fmt.Sscanf(idStr, "%d", &id)
	}
	return id
}
