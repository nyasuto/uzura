package dom

import "testing"

func TestTextNode(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantType NodeType
		wantName string
	}{
		{"basic text", "hello", TextNode, "#text"},
		{"empty text", "", TextNode, "#text"},
		{"text with html", "<b>bold</b>", TextNode, "#text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewText(tt.data)
			if n.NodeType() != tt.wantType {
				t.Errorf("NodeType() = %d, want %d", n.NodeType(), tt.wantType)
			}
			if n.NodeName() != tt.wantName {
				t.Errorf("NodeName() = %q, want %q", n.NodeName(), tt.wantName)
			}
			if n.TextContent() != tt.data {
				t.Errorf("TextContent() = %q, want %q", n.TextContent(), tt.data)
			}
		})
	}
}

func TestTextSetContent(t *testing.T) {
	n := NewText("old")
	n.SetTextContent("new")
	if n.TextContent() != "new" {
		t.Errorf("TextContent() = %q, want %q", n.TextContent(), "new")
	}
}

func TestTextCloneNode(t *testing.T) {
	original := NewText("hello")
	clone := original.CloneNode()

	if clone.Data != original.Data {
		t.Errorf("clone.Data = %q, want %q", clone.Data, original.Data)
	}
	if clone == original {
		t.Error("clone should be a different instance")
	}
	// Modifying clone should not affect original
	clone.Data = "modified"
	if original.Data != "hello" {
		t.Error("modifying clone should not affect original")
	}
}

func TestCommentNode(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantType NodeType
		wantName string
	}{
		{"basic comment", "this is a comment", CommentNode, "#comment"},
		{"empty comment", "", CommentNode, "#comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComment(tt.data)
			if c.NodeType() != tt.wantType {
				t.Errorf("NodeType() = %d, want %d", c.NodeType(), tt.wantType)
			}
			if c.NodeName() != tt.wantName {
				t.Errorf("NodeName() = %q, want %q", c.NodeName(), tt.wantName)
			}
			if c.TextContent() != tt.data {
				t.Errorf("TextContent() = %q, want %q", c.TextContent(), tt.data)
			}
		})
	}
}

func TestCommentSetContent(t *testing.T) {
	c := NewComment("old")
	c.SetTextContent("new")
	if c.TextContent() != "new" {
		t.Errorf("TextContent() = %q, want %q", c.TextContent(), "new")
	}
}

func TestCommentCloneNode(t *testing.T) {
	original := NewComment("hello")
	clone := original.CloneNode()

	if clone.Data != original.Data {
		t.Errorf("clone.Data = %q, want %q", clone.Data, original.Data)
	}
	if clone == original {
		t.Error("clone should be a different instance")
	}
	clone.Data = "modified"
	if original.Data != "hello" {
		t.Error("modifying clone should not affect original")
	}
}
