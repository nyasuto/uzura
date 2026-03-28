// WPT dom/nodes equivalent tests.
// These tests verify compliance with Web Platform Tests for DOM node operations.
package dom

import "testing"

// --- Node properties (WPT: Node-properties.html) ---

func TestWPT_NodeProperties(t *testing.T) {
	doc := NewDocument()

	tests := []struct {
		name     string
		node     Node
		nodeType NodeType
		nodeName string
	}{
		{"Document", doc, DocumentNode, "#document"},
		{"Element", NewElement("div"), ElementNode, "DIV"},
		{"Text", NewText("hello"), TextNode, "#text"},
		{"Comment", NewComment("c"), CommentNode, "#comment"},
		{"DocumentFragment", NewDocumentFragment(), DocumentFragmentNode, "#document-fragment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.NodeType(); got != tt.nodeType {
				t.Errorf("NodeType() = %d, want %d", got, tt.nodeType)
			}
			if got := tt.node.NodeName(); got != tt.nodeName {
				t.Errorf("NodeName() = %q, want %q", got, tt.nodeName)
			}
		})
	}
}

// --- Node.appendChild (WPT: Node-appendChild.html) ---

func TestWPT_AppendChild(t *testing.T) {
	t.Run("append to empty parent", func(t *testing.T) {
		parent := NewElement("div")
		child := NewElement("span")
		result := parent.AppendChild(child)
		if result != Node(child) {
			t.Error("should return appended child")
		}
		if parent.FirstChild() != Node(child) {
			t.Error("child should be first child")
		}
		if child.ParentNode() != Node(parent) {
			t.Error("child parent should be set")
		}
	})

	t.Run("reparenting moves node", func(t *testing.T) {
		p1 := NewElement("div")
		p2 := NewElement("section")
		child := NewElement("span")
		p1.AppendChild(child)
		p2.AppendChild(child)

		if p1.FirstChild() != nil {
			t.Error("old parent should have no children")
		}
		if p2.FirstChild() != Node(child) {
			t.Error("new parent should have child")
		}
	})

	t.Run("append DocumentFragment", func(t *testing.T) {
		parent := NewElement("div")
		frag := NewDocumentFragment()
		a := NewElement("a")
		b := NewElement("b")
		frag.AppendChild(a)
		frag.AppendChild(b)

		parent.AppendChild(frag)

		children := parent.ChildNodes()
		if len(children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(children))
		}
		if children[0] != Node(a) || children[1] != Node(b) {
			t.Error("fragment children should be moved to parent")
		}
		if frag.FirstChild() != nil {
			t.Error("fragment should be empty after append")
		}
	})
}

// --- Node.removeChild (WPT: Node-removeChild.html) ---

func TestWPT_RemoveChild(t *testing.T) {
	t.Run("removes and returns child", func(t *testing.T) {
		parent := NewElement("div")
		child := NewElement("span")
		parent.AppendChild(child)

		removed := parent.RemoveChild(child)
		if removed != Node(child) {
			t.Error("should return removed child")
		}
		if child.ParentNode() != nil {
			t.Error("removed child should have no parent")
		}
		if parent.FirstChild() != nil {
			t.Error("parent should have no children")
		}
	})

	t.Run("sibling pointers updated", func(t *testing.T) {
		parent := NewElement("div")
		a := NewElement("a")
		b := NewElement("b")
		c := NewElement("c")
		parent.AppendChild(a)
		parent.AppendChild(b)
		parent.AppendChild(c)

		parent.RemoveChild(b)

		if a.NextSibling() != Node(c) {
			t.Error("a.NextSibling should be c")
		}
		if c.PreviousSibling() != Node(a) {
			t.Error("c.PreviousSibling should be a")
		}
	})
}

// --- Node.insertBefore (WPT: Node-insertBefore.html) ---

func TestWPT_InsertBefore(t *testing.T) {
	t.Run("insert before existing child", func(t *testing.T) {
		parent := NewElement("div")
		existing := NewElement("b")
		parent.AppendChild(existing)

		newChild := NewElement("a")
		parent.InsertBefore(newChild, existing)

		if parent.FirstChild() != Node(newChild) {
			t.Error("newChild should be first")
		}
		if newChild.NextSibling() != Node(existing) {
			t.Error("newChild.NextSibling should be existing")
		}
	})

	t.Run("nil refChild appends", func(t *testing.T) {
		parent := NewElement("div")
		first := NewElement("a")
		parent.AppendChild(first)

		second := NewElement("b")
		parent.InsertBefore(second, nil)

		if parent.LastChild() != Node(second) {
			t.Error("should append when refChild is nil")
		}
	})
}

// --- Node.replaceChild (WPT: Node-replaceChild.html) ---

func TestWPT_ReplaceChild(t *testing.T) {
	t.Run("replace returns old child", func(t *testing.T) {
		parent := NewElement("div")
		old := NewElement("span")
		parent.AppendChild(old)

		replacement := NewElement("p")
		result := parent.ReplaceChild(replacement, old)
		if result != Node(old) {
			t.Error("should return old child")
		}
		if parent.FirstChild() != Node(replacement) {
			t.Error("parent's child should be replacement")
		}
		if old.ParentNode() != nil {
			t.Error("old child should be detached")
		}
	})

	t.Run("replace with fragment", func(t *testing.T) {
		parent := NewElement("div")
		old := NewElement("span")
		parent.AppendChild(old)

		frag := NewDocumentFragment()
		a := NewElement("a")
		b := NewElement("b")
		frag.AppendChild(a)
		frag.AppendChild(b)

		parent.ReplaceChild(frag, old)

		children := parent.ChildNodes()
		if len(children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(children))
		}
		if children[0] != Node(a) || children[1] != Node(b) {
			t.Error("fragment children should replace old child")
		}
	})
}

// --- Node.cloneNode (WPT: Node-cloneNode.html) ---

func TestWPT_CloneNode(t *testing.T) {
	t.Run("shallow clone has no children", func(t *testing.T) {
		div := NewElement("div")
		div.SetAttribute("id", "test")
		div.AppendChild(NewText("text"))

		clone := div.CloneNode(false).(*Element)
		if clone.GetAttribute("id") != "test" {
			t.Error("attributes should be cloned")
		}
		if clone.FirstChild() != nil {
			t.Error("shallow clone should have no children")
		}
	})

	t.Run("deep clone copies subtree", func(t *testing.T) {
		div := NewElement("div")
		span := NewElement("span")
		span.AppendChild(NewText("hello"))
		div.AppendChild(span)

		clone := div.CloneNode(true).(*Element)
		if clone.FirstChild() == nil {
			t.Fatal("deep clone should have children")
		}
		cloneSpan := clone.FirstChild().(*Element)
		if cloneSpan == span {
			t.Error("cloned child should be a different instance")
		}
		if cloneSpan.FirstChild().(*Text).Data != "hello" {
			t.Error("text content should be cloned")
		}
	})
}

// --- Node.contains (WPT: Node-contains.html) ---

func TestWPT_Contains(t *testing.T) {
	div := NewElement("div")
	span := NewElement("span")
	text := NewText("x")
	div.AppendChild(span)
	span.AppendChild(text)
	detached := NewElement("p")

	if !div.Contains(div) {
		t.Error("node contains itself")
	}
	if !div.Contains(span) {
		t.Error("parent contains child")
	}
	if !div.Contains(text) {
		t.Error("ancestor contains deep descendant")
	}
	if span.Contains(div) {
		t.Error("child does not contain parent")
	}
	if div.Contains(detached) {
		t.Error("does not contain detached node")
	}
	if div.Contains(nil) {
		t.Error("does not contain nil")
	}
}

// --- Element traversal (WPT: ParentNode-children.html, NonDocumentTypeChildNode) ---

func TestWPT_ElementTraversal(t *testing.T) {
	parent := NewElement("div")
	text1 := NewText("before")
	a := NewElement("a")
	text2 := NewText("middle")
	b := NewElement("b")
	text3 := NewText("after")
	parent.AppendChild(text1)
	parent.AppendChild(a)
	parent.AppendChild(text2)
	parent.AppendChild(b)
	parent.AppendChild(text3)

	t.Run("Children", func(t *testing.T) {
		children := parent.Children()
		if len(children) != 2 {
			t.Fatalf("expected 2 element children, got %d", len(children))
		}
		if children[0] != a || children[1] != b {
			t.Error("children should be [a, b]")
		}
	})

	t.Run("FirstElementChild", func(t *testing.T) {
		if parent.FirstElementChild() != a {
			t.Error("firstElementChild should be a")
		}
	})

	t.Run("LastElementChild", func(t *testing.T) {
		if parent.LastElementChild() != b {
			t.Error("lastElementChild should be b")
		}
	})

	t.Run("ChildElementCount", func(t *testing.T) {
		if parent.ChildElementCount() != 2 {
			t.Errorf("childElementCount: got %d, want 2", parent.ChildElementCount())
		}
	})

	t.Run("PreviousElementSibling", func(t *testing.T) {
		if b.PreviousElementSibling() != a {
			t.Error("b.previousElementSibling should be a")
		}
		if a.PreviousElementSibling() != nil {
			t.Error("a.previousElementSibling should be nil")
		}
	})

	t.Run("NextElementSibling", func(t *testing.T) {
		if a.NextElementSibling() != b {
			t.Error("a.nextElementSibling should be b")
		}
		if b.NextElementSibling() != nil {
			t.Error("b.nextElementSibling should be nil")
		}
	})
}

// --- ChildNode.remove (WPT: ChildNode-remove.html) ---

func TestWPT_ChildNodeRemove(t *testing.T) {
	t.Run("Element.Remove", func(t *testing.T) {
		parent := NewElement("div")
		child := NewElement("span")
		parent.AppendChild(child)

		child.Remove()
		if child.ParentNode() != nil {
			t.Error("removed element should have no parent")
		}
		if parent.FirstChild() != nil {
			t.Error("parent should have no children")
		}
	})

	t.Run("Text.Remove", func(t *testing.T) {
		parent := NewElement("div")
		text := NewText("hello")
		parent.AppendChild(text)

		text.Remove()
		if parent.FirstChild() != nil {
			t.Error("parent should have no children")
		}
	})

	t.Run("Comment.Remove", func(t *testing.T) {
		parent := NewElement("div")
		comment := NewComment("x")
		parent.AppendChild(comment)

		comment.Remove()
		if parent.FirstChild() != nil {
			t.Error("parent should have no children")
		}
	})

	t.Run("Remove detached node is no-op", func(t *testing.T) {
		el := NewElement("div")
		el.Remove() // should not panic
	})
}

// --- Document factory methods (WPT: Document-createElement.html etc.) ---

func TestWPT_DocumentFactory(t *testing.T) {
	doc := NewDocument()

	t.Run("createElement sets ownerDocument", func(t *testing.T) {
		el := doc.CreateElement("div")
		if el.OwnerDocument() != doc {
			t.Error("ownerDocument should be set")
		}
		if el.LocalName() != "div" {
			t.Errorf("localName: got %q, want %q", el.LocalName(), "div")
		}
		if el.TagName() != "DIV" {
			t.Errorf("tagName: got %q, want %q", el.TagName(), "DIV")
		}
	})

	t.Run("createTextNode", func(t *testing.T) {
		text := doc.CreateTextNode("test")
		if text.OwnerDocument() != doc {
			t.Error("ownerDocument should be set")
		}
		if text.Data != "test" {
			t.Errorf("data: got %q, want %q", text.Data, "test")
		}
	})

	t.Run("createComment", func(t *testing.T) {
		comment := doc.CreateComment("c")
		if comment.OwnerDocument() != doc {
			t.Error("ownerDocument should be set")
		}
		if comment.Data != "c" {
			t.Errorf("data: got %q, want %q", comment.Data, "c")
		}
	})

	t.Run("createDocumentFragment", func(t *testing.T) {
		frag := doc.CreateDocumentFragment()
		if frag.OwnerDocument() != doc {
			t.Error("ownerDocument should be set")
		}
		if frag.NodeType() != DocumentFragmentNode {
			t.Error("wrong nodeType")
		}
	})
}

// --- Element attributes (WPT: Element-getAttribute.html) ---

func TestWPT_ElementAttributes(t *testing.T) {
	el := NewElement("div")

	if el.GetAttribute("id") != "" {
		t.Error("missing attribute should return empty string")
	}
	if el.HasAttribute("id") {
		t.Error("should not have attribute before set")
	}

	el.SetAttribute("id", "test")
	if el.GetAttribute("id") != "test" {
		t.Error("getAttribute should return set value")
	}
	if !el.HasAttribute("id") {
		t.Error("hasAttribute should return true")
	}

	el.SetAttribute("ID", "upper")
	if el.GetAttribute("id") != "upper" {
		t.Error("attribute names should be case-insensitive")
	}

	el.RemoveAttribute("id")
	if el.HasAttribute("id") {
		t.Error("attribute should be removed")
	}
}

// --- Node.normalize (WPT: Node-normalize.html) ---

func TestWPT_Normalize(t *testing.T) {
	div := NewElement("div")
	div.AppendChild(NewText("a"))
	div.AppendChild(NewText("b"))
	div.AppendChild(NewElement("br"))
	div.AppendChild(NewText(""))
	div.AppendChild(NewText("c"))

	div.Normalize()

	children := div.ChildNodes()
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	if children[0].(*Text).Data != "ab" {
		t.Errorf("first text: got %q, want %q", children[0].(*Text).Data, "ab")
	}
	if _, ok := children[1].(*Element); !ok {
		t.Error("second child should be <br>")
	}
	if children[2].(*Text).Data != "c" {
		t.Errorf("third text: got %q, want %q", children[2].(*Text).Data, "c")
	}
}

// --- Node.isEqualNode (WPT: Node-isEqualNode.html) ---

func TestWPT_IsEqualNode(t *testing.T) {
	t.Run("equal elements", func(t *testing.T) {
		a := NewElement("div")
		a.SetAttribute("class", "x")
		a.AppendChild(NewText("hello"))

		b := NewElement("div")
		b.SetAttribute("class", "x")
		b.AppendChild(NewText("hello"))

		if !a.IsEqualNode(b) {
			t.Error("structurally equal nodes should be equal")
		}
	})

	t.Run("different tag", func(t *testing.T) {
		a := NewElement("div")
		b := NewElement("span")
		if a.IsEqualNode(b) {
			t.Error("different tags should not be equal")
		}
	})

	t.Run("different children", func(t *testing.T) {
		a := NewElement("div")
		a.AppendChild(NewText("a"))
		b := NewElement("div")
		b.AppendChild(NewText("b"))
		if a.IsEqualNode(b) {
			t.Error("different children should not be equal")
		}
	})

	t.Run("nil is not equal", func(t *testing.T) {
		if NewElement("div").IsEqualNode(nil) {
			t.Error("should not be equal to nil")
		}
	})
}

// --- Document element accessors (WPT: Document-doctype.html, Document-body.html) ---

func TestWPT_DocumentAccessors(t *testing.T) {
	doc := NewDocument()
	htmlEl := doc.CreateElement("html")
	head := doc.CreateElement("head")
	title := doc.CreateElement("title")
	title.AppendChild(NewText("Test"))
	head.AppendChild(title)
	body := doc.CreateElement("body")
	htmlEl.AppendChild(head)
	htmlEl.AppendChild(body)
	doc.AppendChild(htmlEl)

	if doc.DocumentElement() != htmlEl {
		t.Error("documentElement should be <html>")
	}
	if doc.Head() != head {
		t.Error("head should return <head>")
	}
	if doc.Body() != body {
		t.Error("body should return <body>")
	}
	if doc.Title() != "Test" {
		t.Errorf("title: got %q, want %q", doc.Title(), "Test")
	}
}

// --- Document.Children (ParentNode mixin on Document) ---

func TestWPT_DocumentChildren(t *testing.T) {
	doc := NewDocument()
	htmlEl := doc.CreateElement("html")
	doc.AppendChild(htmlEl)

	children := doc.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child element, got %d", len(children))
	}
	if children[0] != htmlEl {
		t.Error("child should be <html>")
	}
	if doc.FirstElementChild() != htmlEl {
		t.Error("firstElementChild should be <html>")
	}
	if doc.ChildElementCount() != 1 {
		t.Errorf("childElementCount: got %d, want 1", doc.ChildElementCount())
	}
}

// --- TextContent (WPT: Node-textContent.html) ---

func TestWPT_TextContent(t *testing.T) {
	t.Run("Element textContent", func(t *testing.T) {
		div := NewElement("div")
		div.AppendChild(NewText("hello "))
		span := NewElement("span")
		span.AppendChild(NewText("world"))
		div.AppendChild(span)

		if div.TextContent() != "hello world" {
			t.Errorf("got %q, want %q", div.TextContent(), "hello world")
		}
	})

	t.Run("SetTextContent replaces all", func(t *testing.T) {
		div := NewElement("div")
		div.AppendChild(NewElement("span"))
		div.AppendChild(NewText("old"))

		div.SetTextContent("new")
		children := div.ChildNodes()
		if len(children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(children))
		}
		if children[0].(*Text).Data != "new" {
			t.Error("text content should be replaced")
		}
	})

	t.Run("Document textContent is empty", func(t *testing.T) {
		doc := NewDocument()
		doc.AppendChild(NewElement("html"))
		if doc.TextContent() != "" {
			t.Error("Document.textContent should be empty string")
		}
	})
}

// --- getElementById (WPT: Document-getElementById.html) ---

func TestWPT_GetElementById(t *testing.T) {
	doc := NewDocument()
	div := doc.CreateElement("div")
	div.SetAttribute("id", "target")
	nested := doc.CreateElement("span")
	nested.SetAttribute("id", "deep")
	div.AppendChild(nested)
	doc.AppendChild(div)

	if doc.GetElementById("target") != div {
		t.Error("should find div by id")
	}
	if doc.GetElementById("deep") != nested {
		t.Error("should find nested element by id")
	}
	if doc.GetElementById("missing") != nil {
		t.Error("should return nil for missing id")
	}
}
