package dom

// Text represents a DOM Text node.
type Text struct {
	baseNode
	Data string
}

// NewText creates a new Text node with the given data.
func NewText(data string) *Text {
	return &Text{Data: data}
}

// NodeType returns TextNode.
func (t *Text) NodeType() NodeType {
	return TextNode
}

// NodeName returns "#text".
func (t *Text) NodeName() string {
	return "#text"
}

// AppendChild is not supported for Text nodes; returns nil.
func (t *Text) AppendChild(child Node) Node {
	return nil
}

// RemoveChild is not supported for Text nodes; returns nil.
func (t *Text) RemoveChild(child Node) Node {
	return nil
}

// InsertBefore is not supported for Text nodes; returns nil.
func (t *Text) InsertBefore(newChild, refChild Node) Node {
	return nil
}

// TextContent returns the text data.
func (t *Text) TextContent() string {
	return t.Data
}

// SetTextContent sets the text data.
func (t *Text) SetTextContent(text string) {
	t.Data = text
}

// CloneNode returns a copy of this Text node. The deep parameter is accepted
// for interface compliance but has no effect since Text nodes have no children.
func (t *Text) CloneNode(deep bool) Node {
	clone := NewText(t.Data)
	clone.ownerDocument = t.ownerDocument
	return clone
}

// ReplaceChild is not supported for Text nodes; returns nil.
func (t *Text) ReplaceChild(newChild, oldChild Node) Node {
	return nil
}

// Contains reports whether other is this same node (Text cannot have children).
func (t *Text) Contains(other Node) bool {
	return Node(t) == other
}

// HasChildNodes always returns false for Text nodes.
func (t *Text) HasChildNodes() bool {
	return false
}

// Normalize is a no-op for Text nodes.
func (t *Text) Normalize() {}

// IsEqualNode reports whether other is a Text node with the same data.
func (t *Text) IsEqualNode(other Node) bool {
	return isEqualNode(t, other)
}

// IsSameNode reports whether other is the exact same node reference.
func (t *Text) IsSameNode(other Node) bool {
	return Node(t) == other
}
