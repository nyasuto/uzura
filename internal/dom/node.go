// Package dom implements the DOM tree types following the WHATWG specification.
package dom

// NodeType represents the type of a DOM node.
type NodeType int

const (
	// ElementNode represents an Element node.
	ElementNode NodeType = 1
	// TextNode represents a Text node.
	TextNode NodeType = 3
	// CommentNode represents a Comment node.
	CommentNode NodeType = 8
	// DocumentNode represents a Document node.
	DocumentNode NodeType = 9
)

// Node is the base interface for all DOM nodes.
type Node interface {
	NodeType() NodeType
	NodeName() string
	ParentNode() Node
	ChildNodes() []Node
	FirstChild() Node
	LastChild() Node
	PreviousSibling() Node
	NextSibling() Node
	AppendChild(child Node) Node
	RemoveChild(child Node) Node
	InsertBefore(newChild, refChild Node) Node
	TextContent() string
	SetTextContent(text string)
	OwnerDocument() *Document

	// unexported methods for internal tree manipulation
	setParent(parent Node)
	setPreviousSibling(sibling Node)
	setNextSibling(sibling Node)
	setOwnerDocument(doc *Document)
}

// baseNode provides the shared tree-structure fields for all node types.
type baseNode struct {
	parent        Node
	firstChild    Node
	lastChild     Node
	prevSibling   Node
	nextSibling   Node
	ownerDocument *Document
}

// ParentNode returns the parent of this node.
func (b *baseNode) ParentNode() Node {
	return b.parent
}

// ChildNodes returns a slice of this node's children.
func (b *baseNode) ChildNodes() []Node {
	var children []Node
	for c := b.firstChild; c != nil; c = c.NextSibling() {
		children = append(children, c)
	}
	return children
}

// FirstChild returns the first child of this node.
func (b *baseNode) FirstChild() Node {
	return b.firstChild
}

// LastChild returns the last child of this node.
func (b *baseNode) LastChild() Node {
	return b.lastChild
}

// PreviousSibling returns the previous sibling of this node.
func (b *baseNode) PreviousSibling() Node {
	return b.prevSibling
}

// NextSibling returns the next sibling of this node.
func (b *baseNode) NextSibling() Node {
	return b.nextSibling
}

// OwnerDocument returns the Document that owns this node.
func (b *baseNode) OwnerDocument() *Document {
	return b.ownerDocument
}

func (b *baseNode) setParent(parent Node) {
	b.parent = parent
}

func (b *baseNode) setPreviousSibling(sibling Node) {
	b.prevSibling = sibling
}

func (b *baseNode) setNextSibling(sibling Node) {
	b.nextSibling = sibling
}

func (b *baseNode) setOwnerDocument(doc *Document) {
	b.ownerDocument = doc
}

// appendChild adds child to the end of this node's children list.
// It detaches child from its current parent first.
// Returns the appended child.
func (b *baseNode) appendChild(self, child Node) Node {
	// Detach from old parent
	if p := child.ParentNode(); p != nil {
		p.RemoveChild(child)
	}

	child.setParent(self)
	if b.firstChild == nil {
		b.firstChild = child
	} else {
		b.lastChild.setNextSibling(child)
		child.setPreviousSibling(b.lastChild)
	}
	b.lastChild = child
	return child
}

// removeChild removes child from this node's children list.
// Returns the removed child.
func (b *baseNode) removeChild(child Node) Node {
	prev := child.PreviousSibling()
	next := child.NextSibling()

	if prev != nil {
		prev.setNextSibling(next)
	} else {
		b.firstChild = next
	}

	if next != nil {
		next.setPreviousSibling(prev)
	} else {
		b.lastChild = prev
	}

	child.setParent(nil)
	child.setPreviousSibling(nil)
	child.setNextSibling(nil)
	return child
}

// insertBefore inserts newChild before refChild. If refChild is nil, appends.
func (b *baseNode) insertBefore(self, newChild, refChild Node) Node {
	if refChild == nil {
		return b.appendChild(self, newChild)
	}

	// Detach from old parent
	if p := newChild.ParentNode(); p != nil {
		p.RemoveChild(newChild)
	}

	newChild.setParent(self)
	prev := refChild.PreviousSibling()
	newChild.setPreviousSibling(prev)
	newChild.setNextSibling(refChild)
	refChild.setPreviousSibling(newChild)

	if prev != nil {
		prev.setNextSibling(newChild)
	} else {
		b.firstChild = newChild
	}

	return newChild
}
