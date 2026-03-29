package semantic

import (
	"encoding/json"
	"testing"
)

func TestSemanticNodeCreation(t *testing.T) {
	node := &SemanticNode{
		Role:   "main",
		Name:   "Main Content",
		NodeID: 1,
	}

	if node.Role != "main" {
		t.Errorf("Role = %q, want %q", node.Role, "main")
	}
	if node.Name != "Main Content" {
		t.Errorf("Name = %q, want %q", node.Name, "Main Content")
	}
	if node.NodeID != 1 {
		t.Errorf("NodeID = %d, want %d", node.NodeID, 1)
	}
	if node.Value != "" {
		t.Errorf("Value = %q, want empty", node.Value)
	}
	if len(node.Children) != 0 {
		t.Errorf("Children = %d, want 0", len(node.Children))
	}
}

func TestSemanticNodeWithChildren(t *testing.T) {
	tree := &SemanticNode{
		Role:   "banner",
		Name:   "Site Header",
		NodeID: 1,
		Children: []*SemanticNode{
			{
				Role:   "navigation",
				Name:   "Main Menu",
				NodeID: 2,
				Children: []*SemanticNode{
					{Role: "link", Name: "Home", NodeID: 3, Value: "/"},
					{Role: "link", Name: "About", NodeID: 4, Value: "/about"},
				},
			},
		},
	}

	if tree.Role != "banner" {
		t.Errorf("Role = %q, want %q", tree.Role, "banner")
	}
	if tree.Name != "Site Header" {
		t.Errorf("Name = %q, want %q", tree.Name, "Site Header")
	}
	if tree.NodeID != 1 {
		t.Errorf("NodeID = %d, want %d", tree.NodeID, 1)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("Children = %d, want 1", len(tree.Children))
	}
	nav := tree.Children[0]
	if nav.Role != "navigation" {
		t.Errorf("child Role = %q, want %q", nav.Role, "navigation")
	}
	if len(nav.Children) != 2 {
		t.Fatalf("nav Children = %d, want 2", len(nav.Children))
	}
	if nav.Children[0].Value != "/" {
		t.Errorf("link Value = %q, want %q", nav.Children[0].Value, "/")
	}
}

func TestSemanticNodeJSON(t *testing.T) {
	tree := &SemanticNode{
		Role:   "main",
		Name:   "Content",
		NodeID: 1,
		Children: []*SemanticNode{
			{Role: "heading", Name: "Title", NodeID: 2},
			{Role: "text", Name: "Some text content", NodeID: 3},
			{Role: "link", Name: "Click here", NodeID: 4, Value: "https://example.com"},
		},
	}

	data, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded SemanticNode
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.Role != tree.Role {
		t.Errorf("decoded Role = %q, want %q", decoded.Role, tree.Role)
	}
	if decoded.Name != tree.Name {
		t.Errorf("decoded Name = %q, want %q", decoded.Name, tree.Name)
	}
	if decoded.NodeID != tree.NodeID {
		t.Errorf("decoded NodeID = %d, want %d", decoded.NodeID, tree.NodeID)
	}
	if len(decoded.Children) != 3 {
		t.Fatalf("decoded Children = %d, want 3", len(decoded.Children))
	}
	if decoded.Children[2].Value != "https://example.com" {
		t.Errorf("decoded child Value = %q, want %q", decoded.Children[2].Value, "https://example.com")
	}
}

func TestSemanticNodeJSONOmitEmpty(t *testing.T) {
	node := &SemanticNode{
		Role:   "heading",
		Name:   "Title",
		NodeID: 5,
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal to map failed: %v", err)
	}

	// Value and Children should be omitted when empty
	if _, ok := m["value"]; ok {
		t.Error("empty value should be omitted from JSON")
	}
	if _, ok := m["children"]; ok {
		t.Error("nil children should be omitted from JSON")
	}
}

func TestSemanticNodeFormat(t *testing.T) {
	tree := &SemanticNode{
		Role:   "banner",
		Name:   "Site",
		NodeID: 1,
		Children: []*SemanticNode{
			{
				Role:   "navigation",
				Name:   "Menu",
				NodeID: 2,
				Children: []*SemanticNode{
					{Role: "link", Name: "Home", NodeID: 3, Value: "/"},
					{Role: "link", Name: "About", NodeID: 4, Value: "/about"},
				},
			},
		},
	}

	got := tree.Format(0)
	want := "[banner] Site\n  [navigation] Menu\n    [link#3] Home → /\n    [link#4] About → /about\n"

	if got != want {
		t.Errorf("Format() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSemanticNodeFormatNoValue(t *testing.T) {
	node := &SemanticNode{
		Role:   "heading",
		Name:   "Title",
		NodeID: 5,
	}

	got := node.Format(0)
	want := "[heading] Title\n"

	if got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}
