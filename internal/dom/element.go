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

// TagName returns the tag name in uppercase (alias for NodeName).
func (e *Element) TagName() string {
	return strings.ToUpper(e.localName)
}

// LocalName returns the tag name in lowercase.
func (e *Element) LocalName() string {
	return e.localName
}

// GetAttribute returns the value of the named attribute, or empty string if not present.
func (e *Element) GetAttribute(name string) string {
	name = strings.ToLower(name)
	for _, a := range e.attributes {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// SetAttribute sets the value of the named attribute.
func (e *Element) SetAttribute(name, value string) {
	name = strings.ToLower(name)
	for i, a := range e.attributes {
		if a.Key == name {
			e.attributes[i].Val = value
			return
		}
	}
	e.attributes = append(e.attributes, html.Attribute{Key: name, Val: value})
}

// HasAttribute returns true if the element has the named attribute.
func (e *Element) HasAttribute(name string) bool {
	name = strings.ToLower(name)
	for _, a := range e.attributes {
		if a.Key == name {
			return true
		}
	}
	return false
}

// RemoveAttribute removes the named attribute.
func (e *Element) RemoveAttribute(name string) {
	name = strings.ToLower(name)
	for i, a := range e.attributes {
		if a.Key == name {
			e.attributes = append(e.attributes[:i], e.attributes[i+1:]...)
			return
		}
	}
}

// Attributes returns a copy of this element's attributes.
func (e *Element) Attributes() []html.Attribute {
	cp := make([]html.Attribute, len(e.attributes))
	copy(cp, e.attributes)
	return cp
}

// Id returns the value of the "id" attribute.
func (e *Element) Id() string {
	return e.GetAttribute("id")
}

// ClassName returns the value of the "class" attribute.
func (e *Element) ClassName() string {
	return e.GetAttribute("class")
}

// AppendChild adds a child node to this element.
func (e *Element) AppendChild(child Node) Node {
	return e.appendChild(e, child)
}

// RemoveChild removes a child node from this element.
func (e *Element) RemoveChild(child Node) Node {
	return e.removeChild(child)
}

// InsertBefore inserts newChild before refChild.
func (e *Element) InsertBefore(newChild, refChild Node) Node {
	return e.insertBefore(e, newChild, refChild)
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

// QuerySelector returns the first descendant element matching the CSS selector.
func (e *Element) QuerySelector(sel string) (*Element, error) {
	if SelectorQuery == nil {
		return nil, nil
	}
	return SelectorQuery(e, sel)
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
func (e *Element) QuerySelectorAll(sel string) ([]*Element, error) {
	if SelectorQueryAll == nil {
		return nil, nil
	}
	return SelectorQueryAll(e, sel)
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
