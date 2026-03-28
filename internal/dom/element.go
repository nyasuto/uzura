package dom

import (
	"strings"

	"golang.org/x/net/html"
)

// Element represents a DOM Element node.
type Element struct {
	baseNode
	localName  string
	attributes []html.Attribute
}

// NewElement creates a new Element with the given tag name.
func NewElement(tagName string) *Element {
	return &Element{
		localName: strings.ToLower(tagName),
	}
}

// NodeType returns ElementNode.
func (e *Element) NodeType() NodeType {
	return ElementNode
}

// NodeName returns the tag name in uppercase.
func (e *Element) NodeName() string {
	return strings.ToUpper(e.localName)
}

// AppendChild adds a child node to this element.
func (e *Element) AppendChild(child Node) Node {
	return e.baseNode.appendChild(e, child)
}

// RemoveChild removes a child node from this element.
func (e *Element) RemoveChild(child Node) Node {
	return e.baseNode.removeChild(child)
}

// InsertBefore inserts newChild before refChild.
func (e *Element) InsertBefore(newChild, refChild Node) Node {
	return e.baseNode.insertBefore(e, newChild, refChild)
}

// TextContent returns the concatenated text content of all descendants.
func (e *Element) TextContent() string {
	var sb strings.Builder
	collectText(&sb, e)
	return sb.String()
}

// SetTextContent replaces all children with a single text node.
func (e *Element) SetTextContent(text string) {
	// Remove all children
	for c := e.firstChild; c != nil; {
		next := c.NextSibling()
		e.RemoveChild(c)
		c = next
	}
	if text != "" {
		t := NewText(text)
		e.AppendChild(t)
	}
}

func collectText(sb *strings.Builder, n Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *Text:
			sb.WriteString(v.Data)
		case *Element:
			collectText(sb, v)
		}
	}
}
