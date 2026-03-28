package dom

import "testing"

func TestElementTagName(t *testing.T) {
	tests := []struct {
		input     string
		wantTag   string
		wantLocal string
	}{
		{"div", "DIV", "div"},
		{"DIV", "DIV", "div"},
		{"Span", "SPAN", "span"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			e := NewElement(tt.input)
			if e.TagName() != tt.wantTag {
				t.Errorf("TagName() = %q, want %q", e.TagName(), tt.wantTag)
			}
			if e.LocalName() != tt.wantLocal {
				t.Errorf("LocalName() = %q, want %q", e.LocalName(), tt.wantLocal)
			}
			if e.NodeName() != tt.wantTag {
				t.Errorf("NodeName() = %q, want %q", e.NodeName(), tt.wantTag)
			}
		})
	}
}

func TestElementAttributes(t *testing.T) {
	e := NewElement("div")

	// Initially no attributes
	if e.HasAttribute("id") {
		t.Error("should not have id attribute initially")
	}
	if got := e.GetAttribute("id"); got != "" {
		t.Errorf("GetAttribute(id) = %q, want empty", got)
	}

	// Set attribute
	e.SetAttribute("id", "main")
	if !e.HasAttribute("id") {
		t.Error("should have id attribute after SetAttribute")
	}
	if got := e.GetAttribute("id"); got != "main" {
		t.Errorf("GetAttribute(id) = %q, want %q", got, "main")
	}

	// Update attribute
	e.SetAttribute("id", "updated")
	if got := e.GetAttribute("id"); got != "updated" {
		t.Errorf("GetAttribute(id) = %q, want %q", got, "updated")
	}

	// Remove attribute
	e.RemoveAttribute("id")
	if e.HasAttribute("id") {
		t.Error("should not have id attribute after RemoveAttribute")
	}

	// Remove non-existent attribute (should not panic)
	e.RemoveAttribute("nonexistent")
}

func TestElementAttributeCaseInsensitive(t *testing.T) {
	e := NewElement("div")

	e.SetAttribute("ID", "test")
	if got := e.GetAttribute("id"); got != "test" {
		t.Errorf("GetAttribute(id) = %q, want %q", got, "test")
	}
	if !e.HasAttribute("Id") {
		t.Error("HasAttribute should be case-insensitive")
	}

	e.RemoveAttribute("ID")
	if e.HasAttribute("id") {
		t.Error("RemoveAttribute should be case-insensitive")
	}
}

func TestElementIdAndClassName(t *testing.T) {
	e := NewElement("div")
	e.SetAttribute("id", "myId")
	e.SetAttribute("class", "foo bar")

	if got := e.Id(); got != "myId" {
		t.Errorf("Id() = %q, want %q", got, "myId")
	}
	if got := e.ClassName(); got != "foo bar" {
		t.Errorf("ClassName() = %q, want %q", got, "foo bar")
	}
}

func TestElementTextContent(t *testing.T) {
	div := NewElement("div")
	div.AppendChild(NewText("hello "))
	span := NewElement("span")
	span.AppendChild(NewText("world"))
	div.AppendChild(span)

	if got := div.TextContent(); got != "hello world" {
		t.Errorf("TextContent() = %q, want %q", got, "hello world")
	}

	div.SetTextContent("replaced")
	if got := div.TextContent(); got != "replaced" {
		t.Errorf("after SetTextContent, TextContent() = %q, want %q", got, "replaced")
	}
	if len(div.ChildNodes()) != 1 {
		t.Errorf("after SetTextContent, should have 1 child, got %d", len(div.ChildNodes()))
	}
}

func TestElementSetTextContentEmpty(t *testing.T) {
	div := NewElement("div")
	div.AppendChild(NewText("hello"))
	div.SetTextContent("")
	if len(div.ChildNodes()) != 0 {
		t.Error("SetTextContent('') should remove all children")
	}
}
