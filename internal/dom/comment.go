package dom

// Comment represents a DOM Comment node.
type Comment struct {
	baseNode
	Data string
}

// NewComment creates a new Comment node with the given data.
func NewComment(data string) *Comment {
	return &Comment{Data: data}
}

// NodeType returns CommentNode.
func (c *Comment) NodeType() NodeType {
	return CommentNode
}

// NodeName returns "#comment".
func (c *Comment) NodeName() string {
	return "#comment"
}

// AppendChild is not supported for Comment nodes; returns nil.
func (c *Comment) AppendChild(child Node) Node {
	return nil
}

// RemoveChild is not supported for Comment nodes; returns nil.
func (c *Comment) RemoveChild(child Node) Node {
	return nil
}

// InsertBefore is not supported for Comment nodes; returns nil.
func (c *Comment) InsertBefore(newChild, refChild Node) Node {
	return nil
}

// TextContent returns the comment data.
func (c *Comment) TextContent() string {
	return c.Data
}

// SetTextContent sets the comment data.
func (c *Comment) SetTextContent(text string) {
	oldValue := c.Data
	c.Data = text
	queueCharacterDataMutation(c, oldValue)
}

// CloneNode returns a copy of this Comment node. The deep parameter is accepted
// for interface compliance but has no effect since Comment nodes have no children.
func (c *Comment) CloneNode(deep bool) Node {
	clone := NewComment(c.Data)
	clone.ownerDocument = c.ownerDocument
	return clone
}

// ReplaceChild is not supported for Comment nodes; returns nil.
func (c *Comment) ReplaceChild(newChild, oldChild Node) Node {
	return nil
}

// Contains reports whether other is this same node (Comment cannot have children).
func (c *Comment) Contains(other Node) bool {
	return Node(c) == other
}

// HasChildNodes always returns false for Comment nodes.
func (c *Comment) HasChildNodes() bool {
	return false
}

// Normalize is a no-op for Comment nodes.
func (c *Comment) Normalize() {}

// IsEqualNode reports whether other is a Comment node with the same data.
func (c *Comment) IsEqualNode(other Node) bool {
	return isEqualNode(c, other)
}

// IsSameNode reports whether other is the exact same node reference.
func (c *Comment) IsSameNode(other Node) bool {
	return Node(c) == other
}
