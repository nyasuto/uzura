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

// CloneNode returns a shallow copy of this Text node.
func (t *Text) CloneNode() *Text {
	return NewText(t.Data)
}
