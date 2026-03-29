package semantic

import (
	"strings"
	"testing"

	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func parseHTML(t *testing.T, s string) *SemanticTestResult {
	t.Helper()
	doc, err := htmlparser.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	b := NewBuilder()
	nodes := b.Build(doc)
	return &SemanticTestResult{Nodes: nodes, Builder: b}
}

type SemanticTestResult struct {
	Nodes   []*SemanticNode
	Builder *Builder
}

func TestLandmarkElements(t *testing.T) {
	html := `<html><body>
		<header>Site Header</header>
		<nav>Navigation</nav>
		<main>Main Content</main>
		<aside>Sidebar</aside>
		<footer>Footer</footer>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	expected := []struct {
		role string
		name string
	}{
		{"banner", "Site Header"},
		{"navigation", "Navigation"},
		{"main", "Main Content"},
		{"complementary", "Sidebar"},
		{"contentinfo", "Footer"},
	}

	if len(nodes) != len(expected) {
		t.Fatalf("got %d nodes, want %d", len(nodes), len(expected))
	}

	for i, want := range expected {
		got := nodes[i]
		if got.Role != want.role {
			t.Errorf("nodes[%d].Role = %q, want %q", i, got.Role, want.role)
		}
		if got.Name != want.name {
			t.Errorf("nodes[%d].Name = %q, want %q", i, got.Name, want.name)
		}
		if got.NodeID == 0 {
			t.Errorf("nodes[%d].NodeID should be non-zero", i)
		}
	}
}

func TestArticleAndSection(t *testing.T) {
	html := `<html><body>
		<article>
			<section>Section Content</section>
		</article>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].Role != "article" {
		t.Errorf("Role = %q, want %q", nodes[0].Role, "article")
	}
	if len(nodes[0].Children) != 1 {
		t.Fatalf("children = %d, want 1", len(nodes[0].Children))
	}
	if nodes[0].Children[0].Role != "region" {
		t.Errorf("child Role = %q, want %q", nodes[0].Children[0].Role, "region")
	}
}

func TestARIARoleOverride(t *testing.T) {
	html := `<html><body>
		<div role="navigation">Custom Nav</div>
		<div role="search">Search Box</div>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].Role != "navigation" {
		t.Errorf("nodes[0].Role = %q, want %q", nodes[0].Role, "navigation")
	}
	if nodes[1].Role != "search" {
		t.Errorf("nodes[1].Role = %q, want %q", nodes[1].Role, "search")
	}
}

func TestAriaLabel(t *testing.T) {
	html := `<html><body>
		<nav aria-label="Primary Navigation">Links here</nav>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].Name != "Primary Navigation" {
		t.Errorf("Name = %q, want %q", nodes[0].Name, "Primary Navigation")
	}
}

func TestNoLandmarks(t *testing.T) {
	html := `<html><body>
		<div>
			<p>Just some text</p>
		</div>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0 (no landmarks)", len(nodes))
	}
}

func TestNestedLandmarks(t *testing.T) {
	html := `<html><body>
		<main>
			<nav>Inner Nav</nav>
			<article>
				<header>Article Header</header>
			</article>
		</main>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	main := nodes[0]
	if main.Role != "main" {
		t.Errorf("Role = %q, want %q", main.Role, "main")
	}
	if len(main.Children) != 2 {
		t.Fatalf("main children = %d, want 2", len(main.Children))
	}
	if main.Children[0].Role != "navigation" {
		t.Errorf("child[0].Role = %q, want %q", main.Children[0].Role, "navigation")
	}
	if main.Children[1].Role != "article" {
		t.Errorf("child[1].Role = %q, want %q", main.Children[1].Role, "article")
	}
	// article should have header child
	if len(main.Children[1].Children) != 1 {
		t.Fatalf("article children = %d, want 1", len(main.Children[1].Children))
	}
	if main.Children[1].Children[0].Role != "banner" {
		t.Errorf("article child Role = %q, want %q", main.Children[1].Children[0].Role, "banner")
	}
}

func TestNodeMapPopulated(t *testing.T) {
	html := `<html><body>
		<header>Header</header>
		<main>Main</main>
	</body></html>`

	result := parseHTML(t, html)

	if len(result.Builder.NodeMap) != 2 {
		t.Errorf("NodeMap size = %d, want 2", len(result.Builder.NodeMap))
	}

	for id, elem := range result.Builder.NodeMap {
		if id <= 0 {
			t.Errorf("NodeID %d should be positive", id)
		}
		if elem == nil {
			t.Errorf("NodeMap[%d] should not be nil", id)
		}
	}
}
