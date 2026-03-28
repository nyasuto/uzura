package dom

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// setupParseFragment sets up the HTMLParseFragment function for testing.
func setupParseFragment() {
	HTMLParseFragment = func(parent *Element, htmlStr string) ([]Node, error) {
		r := strings.NewReader(htmlStr)
		context := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Lookup([]byte(parent.LocalName())),
			Data:     parent.LocalName(),
		}
		nodes, err := html.ParseFragment(r, context)
		if err != nil {
			return nil, err
		}
		doc := parent.OwnerDocument()
		if doc == nil {
			doc = NewDocument()
		}
		var result []Node
		for _, n := range nodes {
			result = append(result, convertHTMLNode(doc, n)...)
		}
		return result, nil
	}
}

// convertHTMLNode converts an html.Node tree into dom.Node slice.
func convertHTMLNode(doc *Document, n *html.Node) []Node {
	switch n.Type {
	case html.ElementNode:
		elem := doc.CreateElement(n.Data)
		for _, attr := range n.Attr {
			elem.SetAttribute(attr.Key, attr.Val)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			for _, child := range convertHTMLNode(doc, c) {
				elem.AppendChild(child)
			}
		}
		return []Node{elem}
	case html.TextNode:
		return []Node{doc.CreateTextNode(n.Data)}
	case html.CommentNode:
		return []Node{doc.CreateComment(n.Data)}
	default:
		// For document nodes, process children
		var result []Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			result = append(result, convertHTMLNode(doc, c)...)
		}
		return result
	}
}

func TestSetInnerHTML(t *testing.T) {
	setupParseFragment()

	t.Run("basic HTML", func(t *testing.T) {
		doc := NewDocument()
		div := doc.CreateElement("div")
		doc.AppendChild(div)

		err := div.SetInnerHTML("<span>hello</span>")
		if err != nil {
			t.Fatalf("SetInnerHTML error: %v", err)
		}

		children := div.ChildNodes()
		if len(children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(children))
		}
		span, ok := children[0].(*Element)
		if !ok || span.LocalName() != "span" {
			t.Error("child should be <span>")
		}
		if span.TextContent() != "hello" {
			t.Errorf("text = %q, want %q", span.TextContent(), "hello")
		}
	})

	t.Run("replaces existing children", func(t *testing.T) {
		doc := NewDocument()
		div := doc.CreateElement("div")
		doc.AppendChild(div)
		div.AppendChild(doc.CreateTextNode("old"))

		err := div.SetInnerHTML("<p>new</p>")
		if err != nil {
			t.Fatalf("SetInnerHTML error: %v", err)
		}

		children := div.ChildNodes()
		if len(children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(children))
		}
		if children[0].(*Element).LocalName() != "p" {
			t.Error("child should be <p>")
		}
	})

	t.Run("empty string clears children", func(t *testing.T) {
		doc := NewDocument()
		div := doc.CreateElement("div")
		doc.AppendChild(div)
		div.AppendChild(doc.CreateTextNode("text"))

		err := div.SetInnerHTML("")
		if err != nil {
			t.Fatalf("SetInnerHTML error: %v", err)
		}
		if div.FirstChild() != nil {
			t.Error("should have no children after setting empty innerHTML")
		}
	})

	t.Run("multiple children", func(t *testing.T) {
		doc := NewDocument()
		div := doc.CreateElement("div")
		doc.AppendChild(div)

		err := div.SetInnerHTML("<span>a</span><span>b</span>text")
		if err != nil {
			t.Fatalf("SetInnerHTML error: %v", err)
		}

		children := div.ChildNodes()
		if len(children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(children))
		}
	})

	t.Run("roundtrip innerHTML", func(t *testing.T) {
		doc := NewDocument()
		div := doc.CreateElement("div")
		doc.AppendChild(div)

		input := "<p>hello <strong>world</strong></p>"
		err := div.SetInnerHTML(input)
		if err != nil {
			t.Fatalf("SetInnerHTML error: %v", err)
		}

		got := InnerHTML(div)
		if got != input {
			t.Errorf("roundtrip: got %q, want %q", got, input)
		}
	})

	t.Run("old children detached", func(t *testing.T) {
		doc := NewDocument()
		div := doc.CreateElement("div")
		doc.AppendChild(div)
		old := doc.CreateElement("span")
		div.AppendChild(old)

		err := div.SetInnerHTML("<p>new</p>")
		if err != nil {
			t.Fatalf("SetInnerHTML error: %v", err)
		}

		if old.ParentNode() != nil {
			t.Error("old child should be detached")
		}
	})
}
