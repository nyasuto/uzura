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
