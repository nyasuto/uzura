package semantic

import (
	"testing"
)

func TestRemoveEmptyText(t *testing.T) {
	nodes := []*SemanticNode{
		{Role: "text", Name: "Hello"},
		{Role: "text", Name: "   "},
		{Role: "text", Name: ""},
		{Role: "heading", Name: "Title"},
	}

	result := removeEmptyText(nodes)
	if len(result) != 2 {
		t.Fatalf("got %d nodes, want 2", len(result))
	}
	if result[0].Name != "Hello" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "Hello")
	}
	if result[1].Role != "heading" {
		t.Errorf("result[1].Role = %q, want %q", result[1].Role, "heading")
	}
}

func TestMergeAdjacentText(t *testing.T) {
	nodes := []*SemanticNode{
		{Role: "text", Name: "First"},
		{Role: "text", Name: "Second"},
		{Role: "text", Name: "Third"},
		{Role: "heading", Name: "Title"},
		{Role: "text", Name: "After heading"},
	}

	result := mergeAdjacentText(nodes)
	if len(result) != 3 {
		t.Fatalf("got %d nodes, want 3", len(result))
	}
	if result[0].Name != "First Second Third" {
		t.Errorf("merged Name = %q, want %q", result[0].Name, "First Second Third")
	}
	if result[1].Role != "heading" {
		t.Errorf("result[1].Role = %q, want %q", result[1].Role, "heading")
	}
	if result[2].Name != "After heading" {
		t.Errorf("result[2].Name = %q, want %q", result[2].Name, "After heading")
	}
}

func TestEnforceDepth(t *testing.T) {
	deep := &SemanticNode{
		Role: "main",
		Name: "Root",
		Children: []*SemanticNode{
			{
				Role: "region",
				Name: "Level 1",
				Children: []*SemanticNode{
					{
						Role: "region",
						Name: "Level 2",
						Children: []*SemanticNode{
							{Role: "text", Name: "Level 3"},
						},
					},
				},
			},
		},
	}

	result := enforceDepth([]*SemanticNode{deep}, 0, 3)
	if len(result) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result))
	}
	level1 := result[0].Children
	if len(level1) != 1 {
		t.Fatalf("level1 = %d, want 1", len(level1))
	}
	level2 := level1[0].Children
	if len(level2) != 1 {
		t.Fatalf("level2 = %d, want 1", len(level2))
	}
	// Level 3 children should be truncated
	if len(level2[0].Children) != 0 {
		t.Errorf("level3 children = %d, want 0 (truncated)", len(level2[0].Children))
	}
}

func TestEnforceDepthShallow(t *testing.T) {
	nodes := []*SemanticNode{
		{
			Role: "main",
			Children: []*SemanticNode{
				{Role: "heading", Name: "Title"},
			},
		},
	}

	result := enforceDepth(nodes, 0, 1)
	if len(result) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result))
	}
	if len(result[0].Children) != 0 {
		t.Errorf("children = %d, want 0 (depth=1 should cut children)", len(result[0].Children))
	}
}

func TestHiddenElements(t *testing.T) {
	html := `<html><body>
		<main>
			<h1>Visible Title</h1>
			<div hidden><p>Hidden content</p></div>
			<div aria-hidden="true"><p>Also hidden</p></div>
			<h2>Visible Subtitle</h2>
		</main>
	</body></html>`

	result := parseHTML(t, html)
	compressed := CompressTree(result.Builder, result.Nodes, DefaultMaxDepth)

	if len(compressed) != 1 {
		t.Fatalf("got %d nodes, want 1", len(compressed))
	}
	main := compressed[0]
	// Hidden elements should not appear at all. Only visible headings remain.
	if len(main.Children) != 2 {
		t.Fatalf("main children = %d, want 2", len(main.Children))
	}
	if main.Children[0].Name != "Visible Title" {
		t.Errorf("child[0].Name = %q, want %q", main.Children[0].Name, "Visible Title")
	}
	if main.Children[1].Name != "Visible Subtitle" {
		t.Errorf("child[1].Name = %q, want %q", main.Children[1].Name, "Visible Subtitle")
	}
}

func TestCompressFullPipeline(t *testing.T) {
	html := `<html><body>
		<main>
			<h1>Title</h1>
			<div hidden>Secret</div>
			<p>First paragraph</p>
			<p>Second paragraph</p>
		</main>
	</body></html>`

	result := parseHTML(t, html)
	before := countNodes(result.Nodes)
	compressed := CompressTree(result.Builder, result.Nodes, DefaultMaxDepth)
	after := countNodes(compressed)

	// Compressed tree should be no larger than original
	if after > before {
		t.Errorf("compressed size %d > original size %d", after, before)
	}
}

func countNodes(nodes []*SemanticNode) int {
	count := len(nodes)
	for _, n := range nodes {
		count += countNodes(n.Children)
	}
	return count
}
