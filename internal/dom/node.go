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
	// DocumentFragmentNode represents a DocumentFragment node.
	DocumentFragmentNode NodeType = 11
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
	ReplaceChild(newChild, oldChild Node) Node
	CloneNode(deep bool) Node
	TextContent() string
	SetTextContent(text string)
	OwnerDocument() *Document
	Contains(other Node) bool
	HasChildNodes() bool
	Normalize()
	IsEqualNode(other Node) bool
	IsSameNode(other Node) bool

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
// If child is a DocumentFragment, its children are moved instead.
// Returns the appended child (or the fragment).
func (b *baseNode) appendChild(self, child Node) Node {
	if frag, ok := child.(*DocumentFragment); ok {
		for c := frag.firstChild; c != nil; {
			next := c.NextSibling()
			frag.removeChild(c)
			b.appendSingleChild(self, c)
			c = next
		}
		return child
	}
	b.appendSingleChild(self, child)
	return child
}

// appendSingleChild appends a single (non-fragment) child.
func (b *baseNode) appendSingleChild(self, child Node) {
	if p := child.ParentNode(); p != nil {
		p.RemoveChild(child)
	}
	prevSibling := b.lastChild
	child.setParent(self)
	if b.firstChild == nil {
		b.firstChild = child
	} else {
		b.lastChild.setNextSibling(child)
		child.setPreviousSibling(b.lastChild)
	}
	b.lastChild = child
	queueChildListMutation(self, []Node{child}, nil, prevSibling, nil)
}

// removeChild removes child from this node's children list.
// Returns the removed child.
func (b *baseNode) removeChild(child Node) Node {
	target := child.ParentNode()
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
	queueChildListMutation(target, nil, []Node{child}, prev, next)
	return child
}

// replaceChild replaces oldChild with newChild in this node's children.
// If newChild is a DocumentFragment, oldChild is replaced by the fragment's children.
func (b *baseNode) replaceChild(self, newChild, oldChild Node) Node {
	if frag, ok := newChild.(*DocumentFragment); ok {
		ref := oldChild.NextSibling()
		b.removeChild(oldChild)
		for c := frag.firstChild; c != nil; {
			next := c.NextSibling()
			frag.removeChild(c)
			if ref != nil {
				b.insertSingleBefore(self, c, ref)
			} else {
				b.appendSingleChild(self, c)
			}
			c = next
		}
		return oldChild
	}

	if p := newChild.ParentNode(); p != nil {
		p.RemoveChild(newChild)
	}

	newChild.setParent(self)
	prev := oldChild.PreviousSibling()
	next := oldChild.NextSibling()

	newChild.setPreviousSibling(prev)
	newChild.setNextSibling(next)

	if prev != nil {
		prev.setNextSibling(newChild)
	} else {
		b.firstChild = newChild
	}

	if next != nil {
		next.setPreviousSibling(newChild)
	} else {
		b.lastChild = newChild
	}

	oldChild.setParent(nil)
	oldChild.setPreviousSibling(nil)
	oldChild.setNextSibling(nil)
	queueChildListMutation(self, []Node{newChild}, []Node{oldChild}, prev, next)
	return oldChild
}

// contains reports whether other is a descendant of this node or is this node itself.
func (b *baseNode) contains(self, other Node) bool {
	if other == nil {
		return false
	}
	if self == other {
		return true
	}
	for c := b.firstChild; c != nil; c = c.NextSibling() {
		if c.Contains(other) {
			return true
		}
	}
	return false
}

// hasChildNodes reports whether this node has any children.
func (b *baseNode) hasChildNodes() bool {
	return b.firstChild != nil
}

// normalize merges adjacent Text nodes and removes empty Text nodes.
func (b *baseNode) normalize(self Node) {
	var next Node
	for c := b.firstChild; c != nil; c = next {
		next = c.NextSibling()
		if t, ok := c.(*Text); ok {
			if t.Data == "" {
				self.RemoveChild(c)
				continue
			}
			// Merge consecutive text nodes
			for next != nil {
				nextText, ok := next.(*Text)
				if !ok {
					break
				}
				t.Data += nextText.Data
				afterNext := next.NextSibling()
				self.RemoveChild(next)
				next = afterNext
			}
		} else {
			c.Normalize()
		}
	}
}

// isSameNode reports whether other is the exact same node (identity check).
func (b *baseNode) isSameNode(self, other Node) bool {
	return self == other
}

// insertBefore inserts newChild before refChild. If refChild is nil, appends.
// If newChild is a DocumentFragment, its children are inserted instead.
func (b *baseNode) insertBefore(self, newChild, refChild Node) Node {
	if refChild == nil {
		return b.appendChild(self, newChild)
	}

	if frag, ok := newChild.(*DocumentFragment); ok {
		for c := frag.firstChild; c != nil; {
			next := c.NextSibling()
			frag.removeChild(c)
			b.insertSingleBefore(self, c, refChild)
			c = next
		}
		return newChild
	}

	b.insertSingleBefore(self, newChild, refChild)
	return newChild
}

// insertSingleBefore inserts a single (non-fragment) node before refChild.
func (b *baseNode) insertSingleBefore(self, newChild, refChild Node) {
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
	queueChildListMutation(self, []Node{newChild}, nil, prev, refChild)
}
