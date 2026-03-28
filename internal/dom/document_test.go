package dom

import "testing"

// buildSimpleDoc creates: <html><head><title>T</title></head><body>...</body></html>
func buildSimpleDoc(title string) *Document {
	doc := NewDocument()
	html := doc.CreateElement("html")
	head := doc.CreateElement("head")
	body := doc.CreateElement("body")
	titleElem := doc.CreateElement("title")
	titleElem.AppendChild(doc.CreateTextNode(title))
	head.AppendChild(titleElem)
	html.AppendChild(head)
	html.AppendChild(body)
	doc.AppendChild(html)
	return doc
}

func TestDocumentStructure(t *testing.T) {
	doc := buildSimpleDoc("Hello")

	if doc.DocumentElement() == nil {
		t.Fatal("DocumentElement() should not be nil")
	}
	if doc.DocumentElement().LocalName() != "html" {
		t.Errorf("DocumentElement().LocalName() = %q, want %q", doc.DocumentElement().LocalName(), "html")
	}
	if doc.Head() == nil {
		t.Fatal("Head() should not be nil")
	}
	if doc.Body() == nil {
		t.Fatal("Body() should not be nil")
	}
	if got := doc.Title(); got != "Hello" {
		t.Errorf("Title() = %q, want %q", got, "Hello")
	}
}

func TestDocumentCreateElement(t *testing.T) {
	doc := NewDocument()
	e := doc.CreateElement("div")

	if e.OwnerDocument() != doc {
		t.Error("created element should have ownerDocument set")
	}
	if e.NodeType() != ElementNode {
		t.Errorf("NodeType() = %d, want %d", e.NodeType(), ElementNode)
	}
}

func TestDocumentCreateTextNode(t *testing.T) {
	doc := NewDocument()
	t2 := doc.CreateTextNode("hello")
	if t2.OwnerDocument() != doc {
		t.Error("created text should have ownerDocument set")
	}
	if t2.Data != "hello" {
		t.Errorf("Data = %q, want %q", t2.Data, "hello")
	}
}

func TestDocumentCreateComment(t *testing.T) {
	doc := NewDocument()
	c := doc.CreateComment("note")
	if c.OwnerDocument() != doc {
		t.Error("created comment should have ownerDocument set")
	}
	if c.Data != "note" {
		t.Errorf("Data = %q, want %q", c.Data, "note")
	}
}

func TestDocumentGetElementById(t *testing.T) {
	doc := buildSimpleDoc("Test")
	body := doc.Body()

	div := doc.CreateElement("div")
	div.SetAttribute("id", "main")
	body.AppendChild(div)

	nested := doc.CreateElement("span")
	nested.SetAttribute("id", "nested")
	div.AppendChild(nested)

	tests := []struct {
		id   string
		want string // expected tag name, empty if nil
	}{
		{"main", "DIV"},
		{"nested", "SPAN"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			found := doc.GetElementById(tt.id)
			if tt.want == "" {
				if found != nil {
					t.Errorf("expected nil, got %q", found.TagName())
				}
			} else {
				if found == nil {
					t.Fatalf("expected %q, got nil", tt.want)
				}
				if found.TagName() != tt.want {
					t.Errorf("TagName() = %q, want %q", found.TagName(), tt.want)
				}
			}
		})
	}
}

func TestDocumentGetElementsByTagName(t *testing.T) {
	doc := buildSimpleDoc("Test")
	body := doc.Body()

	body.AppendChild(doc.CreateElement("div"))
	body.AppendChild(doc.CreateElement("div"))
	body.AppendChild(doc.CreateElement("span"))

	divs := doc.GetElementsByTagName("div")
	if len(divs) != 2 {
		t.Errorf("GetElementsByTagName(div) returned %d, want 2", len(divs))
	}

	all := doc.GetElementsByTagName("*")
	// expected: html, head, title, body, div, div, span
	if len(all) != 7 {
		t.Errorf("GetElementsByTagName(*) returned %d, want 7", len(all))
	}
}

func TestDocumentGetElementsByClassName(t *testing.T) {
	doc := buildSimpleDoc("Test")
	body := doc.Body()

	d1 := doc.CreateElement("div")
	d1.SetAttribute("class", "foo bar")
	body.AppendChild(d1)

	d2 := doc.CreateElement("div")
	d2.SetAttribute("class", "foo")
	body.AppendChild(d2)

	d3 := doc.CreateElement("div")
	d3.SetAttribute("class", "bar baz")
	body.AppendChild(d3)

	foos := doc.GetElementsByClassName("foo")
	if len(foos) != 2 {
		t.Errorf("GetElementsByClassName(foo) returned %d, want 2", len(foos))
	}

	fooBar := doc.GetElementsByClassName("foo bar")
	if len(fooBar) != 1 {
		t.Errorf("GetElementsByClassName(foo bar) returned %d, want 1", len(fooBar))
	}

	empty := doc.GetElementsByClassName("")
	if len(empty) != 0 {
		t.Errorf("GetElementsByClassName('') returned %d, want 0", len(empty))
	}
}

func TestDocumentTextContent(t *testing.T) {
	doc := NewDocument()
	if doc.TextContent() != "" {
		t.Error("Document.TextContent() should return empty string")
	}
	doc.SetTextContent("ignored") // should be no-op
}
