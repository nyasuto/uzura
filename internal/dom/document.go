package dom

// Document represents a DOM Document node.
type Document struct {
	baseNode
}

// NewDocument creates a new empty Document.
func NewDocument() *Document {
	return &Document{}
}

// NodeType returns DocumentNode.
func (d *Document) NodeType() NodeType {
	return DocumentNode
}

// NodeName returns "#document".
func (d *Document) NodeName() string {
	return "#document"
}

// AppendChild adds a child node to this document.
func (d *Document) AppendChild(child Node) Node {
	return d.baseNode.appendChild(d, child)
}

// RemoveChild removes a child node from this document.
func (d *Document) RemoveChild(child Node) Node {
	return d.baseNode.removeChild(child)
}

// InsertBefore inserts newChild before refChild.
func (d *Document) InsertBefore(newChild, refChild Node) Node {
	return d.baseNode.insertBefore(d, newChild, refChild)
}

// TextContent for Document always returns empty string per spec.
func (d *Document) TextContent() string {
	return ""
}

// SetTextContent for Document is a no-op per spec.
func (d *Document) SetTextContent(text string) {}
