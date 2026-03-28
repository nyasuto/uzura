package dom

import "testing"

func TestCloneNodeShallow(t *testing.T) {
	t.Run("Text", func(t *testing.T) {
		orig := NewText("hello")
		clone := orig.CloneNode(false).(*Text)
		if clone.Data != "hello" {
			t.Errorf("got %q, want %q", clone.Data, "hello")
		}
		if clone.ParentNode() != nil {
			t.Error("clone should have no parent")
		}
	})

	t.Run("Comment", func(t *testing.T) {
		orig := NewComment("a comment")
		clone := orig.CloneNode(false).(*Comment)
		if clone.Data != "a comment" {
			t.Errorf("got %q, want %q", clone.Data, "a comment")
		}
	})

	t.Run("Element shallow does not copy children", func(t *testing.T) {
		div := NewElement("div")
		div.SetAttribute("id", "main")
		div.SetAttribute("class", "container")
		div.AppendChild(NewText("child text"))

		clone := div.CloneNode(false).(*Element)
		if clone.LocalName() != "div" {
			t.Errorf("localName: got %q, want %q", clone.LocalName(), "div")
		}
		if clone.GetAttribute("id") != "main" {
			t.Errorf("id: got %q, want %q", clone.GetAttribute("id"), "main")
		}
		if clone.GetAttribute("class") != "container" {
			t.Errorf("class: got %q, want %q", clone.GetAttribute("class"), "container")
		}
		if clone.FirstChild() != nil {
			t.Error("shallow clone should have no children")
		}
		if clone.ParentNode() != nil {
			t.Error("clone should have no parent")
		}
	})

	t.Run("Element clone attributes are independent", func(t *testing.T) {
		div := NewElement("div")
		div.SetAttribute("id", "orig")
		clone := div.CloneNode(false).(*Element)
		clone.SetAttribute("id", "cloned")
		if div.GetAttribute("id") != "orig" {
			t.Error("modifying clone should not affect original")
		}
	})
}

func TestCloneNodeDeep(t *testing.T) {
	// Build: <div id="root"><span>hello</span><!-- comment --></div>
	div := NewElement("div")
	div.SetAttribute("id", "root")
	span := NewElement("span")
	span.AppendChild(NewText("hello"))
	div.AppendChild(span)
	div.AppendChild(NewComment("comment"))

	clone := div.CloneNode(true).(*Element)

	if clone.GetAttribute("id") != "root" {
		t.Error("root attribute not cloned")
	}

	children := clone.ChildNodes()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}

	clonedSpan, ok := children[0].(*Element)
	if !ok || clonedSpan.LocalName() != "span" {
		t.Error("first child should be <span>")
	}

	spanChildren := clonedSpan.ChildNodes()
	if len(spanChildren) != 1 {
		t.Fatalf("span should have 1 child, got %d", len(spanChildren))
	}
	clonedText, ok := spanChildren[0].(*Text)
	if !ok || clonedText.Data != "hello" {
		t.Error("text node not cloned correctly")
	}

	clonedComment, ok := children[1].(*Comment)
	if !ok || clonedComment.Data != "comment" {
		t.Error("comment not cloned correctly")
	}

	// Verify independence: modify clone, original unchanged
	clonedSpan.SetAttribute("class", "new")
	if span.HasAttribute("class") {
		t.Error("modifying deep clone should not affect original")
	}
}

func TestReplaceChild(t *testing.T) {
	parent := NewElement("div")
	oldChild := NewElement("span")
	newChild := NewElement("p")
	extra := NewElement("a")

	parent.AppendChild(oldChild)
	parent.AppendChild(extra)

	replaced := parent.ReplaceChild(newChild, oldChild)
	if replaced != oldChild {
		t.Error("ReplaceChild should return old child")
	}
	if oldChild.ParentNode() != nil {
		t.Error("old child should be detached")
	}

	children := parent.ChildNodes()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0] != Node(newChild) {
		t.Error("first child should be newChild")
	}
	if children[1] != Node(extra) {
		t.Error("second child should be extra")
	}

	// Sibling pointers
	if newChild.NextSibling() != Node(extra) {
		t.Error("newChild.NextSibling should be extra")
	}
	if extra.PreviousSibling() != Node(newChild) {
		t.Error("extra.PreviousSibling should be newChild")
	}
}

func TestReplaceChildOnlyChild(t *testing.T) {
	parent := NewElement("div")
	old := NewText("old")
	parent.AppendChild(old)

	replacement := NewText("new")
	parent.ReplaceChild(replacement, old)

	if parent.FirstChild() != Node(replacement) {
		t.Error("firstChild should be replacement")
	}
	if parent.LastChild() != Node(replacement) {
		t.Error("lastChild should be replacement")
	}
}

func TestContains(t *testing.T) {
	// <div><span><a>text</a></span></div>
	div := NewElement("div")
	span := NewElement("span")
	a := NewElement("a")
	text := NewText("text")
	div.AppendChild(span)
	span.AppendChild(a)
	a.AppendChild(text)

	tests := []struct {
		name  string
		node  Node
		other Node
		want  bool
	}{
		{"self", div, div, true},
		{"direct child", div, span, true},
		{"deep descendant", div, text, true},
		{"not descendant", span, div, false},
		{"sibling not contained", a, span, false},
		{"nil", div, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bool
			switch n := tt.node.(type) {
			case *Element:
				got = n.Contains(tt.other)
			case *Document:
				got = n.Contains(tt.other)
			}
			if got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasChildNodes(t *testing.T) {
	div := NewElement("div")
	if div.HasChildNodes() {
		t.Error("empty element should return false")
	}
	div.AppendChild(NewText("hi"))
	if !div.HasChildNodes() {
		t.Error("element with children should return true")
	}
}

func TestNormalize(t *testing.T) {
	t.Run("merge adjacent text nodes", func(t *testing.T) {
		div := NewElement("div")
		div.AppendChild(NewText("hello"))
		div.AppendChild(NewText(" "))
		div.AppendChild(NewText("world"))

		div.Normalize()

		children := div.ChildNodes()
		if len(children) != 1 {
			t.Fatalf("expected 1 child after normalize, got %d", len(children))
		}
		if children[0].(*Text).Data != "hello world" {
			t.Errorf("got %q, want %q", children[0].(*Text).Data, "hello world")
		}
	})

	t.Run("remove empty text nodes", func(t *testing.T) {
		div := NewElement("div")
		div.AppendChild(NewText(""))
		div.AppendChild(NewElement("span"))
		div.AppendChild(NewText(""))

		div.Normalize()

		children := div.ChildNodes()
		if len(children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(children))
		}
		if _, ok := children[0].(*Element); !ok {
			t.Error("remaining child should be the span element")
		}
	})

	t.Run("mixed content", func(t *testing.T) {
		div := NewElement("div")
		div.AppendChild(NewText("a"))
		div.AppendChild(NewText("b"))
		div.AppendChild(NewElement("br"))
		div.AppendChild(NewText("c"))
		div.AppendChild(NewText("d"))

		div.Normalize()

		children := div.ChildNodes()
		if len(children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(children))
		}
		if children[0].(*Text).Data != "ab" {
			t.Errorf("first text: got %q, want %q", children[0].(*Text).Data, "ab")
		}
		if children[2].(*Text).Data != "cd" {
			t.Errorf("last text: got %q, want %q", children[2].(*Text).Data, "cd")
		}
	})

	t.Run("recursive normalization", func(t *testing.T) {
		div := NewElement("div")
		span := NewElement("span")
		span.AppendChild(NewText("x"))
		span.AppendChild(NewText("y"))
		div.AppendChild(span)

		div.Normalize()

		spanChildren := span.ChildNodes()
		if len(spanChildren) != 1 {
			t.Fatalf("expected 1 child in span, got %d", len(spanChildren))
		}
		if spanChildren[0].(*Text).Data != "xy" {
			t.Errorf("got %q, want %q", spanChildren[0].(*Text).Data, "xy")
		}
	})
}

func TestIsEqualNode(t *testing.T) {
	t.Run("equal text nodes", func(t *testing.T) {
		a := NewText("hello")
		b := NewText("hello")
		if !a.IsEqualNode(b) {
			t.Error("identical text nodes should be equal")
		}
	})

	t.Run("different text nodes", func(t *testing.T) {
		a := NewText("hello")
		b := NewText("world")
		if a.IsEqualNode(b) {
			t.Error("different text nodes should not be equal")
		}
	})

	t.Run("equal elements with attributes", func(t *testing.T) {
		a := NewElement("div")
		a.SetAttribute("id", "x")
		a.SetAttribute("class", "y")
		b := NewElement("div")
		b.SetAttribute("id", "x")
		b.SetAttribute("class", "y")
		if !a.IsEqualNode(b) {
			t.Error("identical elements should be equal")
		}
	})

	t.Run("different attributes", func(t *testing.T) {
		a := NewElement("div")
		a.SetAttribute("id", "x")
		b := NewElement("div")
		b.SetAttribute("id", "y")
		if a.IsEqualNode(b) {
			t.Error("elements with different attributes should not be equal")
		}
	})

	t.Run("deep equality with children", func(t *testing.T) {
		a := NewElement("div")
		a.AppendChild(NewText("hello"))
		b := NewElement("div")
		b.AppendChild(NewText("hello"))
		if !a.IsEqualNode(b) {
			t.Error("elements with equal children should be equal")
		}
	})

	t.Run("different children", func(t *testing.T) {
		a := NewElement("div")
		a.AppendChild(NewText("hello"))
		b := NewElement("div")
		a.AppendChild(NewText("world"))
		if a.IsEqualNode(b) {
			t.Error("elements with different children should not be equal")
		}
	})

	t.Run("nil comparison", func(t *testing.T) {
		a := NewText("hi")
		if a.IsEqualNode(nil) {
			t.Error("should not be equal to nil")
		}
	})

	t.Run("equal comments", func(t *testing.T) {
		a := NewComment("test")
		b := NewComment("test")
		if !a.IsEqualNode(b) {
			t.Error("identical comments should be equal")
		}
	})
}

func TestIsSameNode(t *testing.T) {
	a := NewElement("div")
	b := NewElement("div")

	if !a.IsSameNode(a) {
		t.Error("same reference should return true")
	}
	if a.IsSameNode(b) {
		t.Error("different references should return false")
	}
}

func TestCloneNodeDocument(t *testing.T) {
	doc := NewDocument()
	htmlEl := NewElement("html")
	htmlEl.AppendChild(NewElement("head"))
	htmlEl.AppendChild(NewElement("body"))
	doc.AppendChild(htmlEl)

	clone := doc.CloneNode(true).(*Document)
	if clone.DocumentElement() == nil {
		t.Fatal("cloned document should have html element")
	}
	if clone.Head() == nil {
		t.Error("cloned document should have head")
	}
	if clone.Body() == nil {
		t.Error("cloned document should have body")
	}
	// Independence
	if clone.DocumentElement() == htmlEl {
		t.Error("cloned elements should be different instances")
	}
}
