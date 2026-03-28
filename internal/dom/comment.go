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
	c.Data = text
}

// CloneNode returns a shallow copy of this Comment.
func (c *Comment) CloneNode() *Comment {
	return NewComment(c.Data)
}
