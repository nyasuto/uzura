package dom

import "strings"

// DocumentFragment represents a lightweight document container.
// When inserted into the DOM, its children are inserted instead of the fragment itself.
type DocumentFragment struct {
	baseNode
}

// NewDocumentFragment creates a new empty DocumentFragment.
func NewDocumentFragment() *DocumentFragment {
	return &DocumentFragment{}
}

// NodeType returns DocumentFragmentNode.
func (f *DocumentFragment) NodeType() NodeType {
	return DocumentFragmentNode
}

// NodeName returns "#document-fragment".
func (f *DocumentFragment) NodeName() string {
	return "#document-fragment"
}

// AppendChild adds a child node to this fragment.
func (f *DocumentFragment) AppendChild(child Node) Node {
	return f.appendChild(f, child)
}

// RemoveChild removes a child node from this fragment.
func (f *DocumentFragment) RemoveChild(child Node) Node {
	return f.removeChild(child)
}

// InsertBefore inserts newChild before refChild.
func (f *DocumentFragment) InsertBefore(newChild, refChild Node) Node {
	return f.insertBefore(f, newChild, refChild)
}

// ReplaceChild replaces oldChild with newChild. Returns oldChild.
func (f *DocumentFragment) ReplaceChild(newChild, oldChild Node) Node {
	return f.replaceChild(f, newChild, oldChild)
}

// CloneNode returns a copy of this fragment. If deep is true, all descendants are also cloned.
func (f *DocumentFragment) CloneNode(deep bool) Node {
	clone := NewDocumentFragment()
	clone.ownerDocument = f.ownerDocument
	if deep {
		for c := f.firstChild; c != nil; c = c.NextSibling() {
			clone.AppendChild(c.CloneNode(true))
		}
	}
	return clone
}

// Contains reports whether other is a descendant of this fragment (or is itself).
func (f *DocumentFragment) Contains(other Node) bool {
	return f.contains(f, other)
}

// HasChildNodes reports whether this fragment has any children.
func (f *DocumentFragment) HasChildNodes() bool {
	return f.hasChildNodes()
}

// Normalize merges adjacent Text nodes and removes empty Text nodes.
func (f *DocumentFragment) Normalize() {
	f.normalize(f)
}

// IsEqualNode reports whether other is structurally equal to this fragment.
func (f *DocumentFragment) IsEqualNode(other Node) bool {
	return isEqualNode(f, other)
}

// IsSameNode reports whether other is the exact same node reference.
func (f *DocumentFragment) IsSameNode(other Node) bool {
	return f.isSameNode(f, other)
}

// TextContent returns the concatenated text content of all descendants.
func (f *DocumentFragment) TextContent() string {
	var sb strings.Builder
	collectText(&sb, f)
	return sb.String()
}

// SetTextContent replaces all children with a single text node.
func (f *DocumentFragment) SetTextContent(text string) {
	for c := f.firstChild; c != nil; {
		next := c.NextSibling()
		f.RemoveChild(c)
		c = next
	}
	if text != "" {
		t := NewText(text)
		f.AppendChild(t)
	}
}

// QuerySelector returns the first descendant element matching the CSS selector.
func (f *DocumentFragment) QuerySelector(sel string) (*Element, error) {
	if SelectorQuery == nil {
		return nil, nil
	}
	return SelectorQuery(f, sel)
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
func (f *DocumentFragment) QuerySelectorAll(sel string) ([]*Element, error) {
	if SelectorQueryAll == nil {
		return nil, nil
	}
	return SelectorQueryAll(f, sel)
}
