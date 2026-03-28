package dom

// PreviousElementSibling returns the previous sibling that is an Element, or nil.
func (e *Element) PreviousElementSibling() *Element {
	for s := e.prevSibling; s != nil; s = s.PreviousSibling() {
		if el, ok := s.(*Element); ok {
			return el
		}
	}
	return nil
}

// NextElementSibling returns the next sibling that is an Element, or nil.
func (e *Element) NextElementSibling() *Element {
	for s := e.nextSibling; s != nil; s = s.NextSibling() {
		if el, ok := s.(*Element); ok {
			return el
		}
	}
	return nil
}

// Children returns all child Elements of this element.
func (e *Element) Children() []*Element {
	return collectChildElements(e)
}

// FirstElementChild returns the first child that is an Element, or nil.
func (e *Element) FirstElementChild() *Element {
	for c := e.firstChild; c != nil; c = c.NextSibling() {
		if el, ok := c.(*Element); ok {
			return el
		}
	}
	return nil
}

// LastElementChild returns the last child that is an Element, or nil.
func (e *Element) LastElementChild() *Element {
	for c := e.lastChild; c != nil; c = c.PreviousSibling() {
		if el, ok := c.(*Element); ok {
			return el
		}
	}
	return nil
}

// ChildElementCount returns the number of child Elements.
func (e *Element) ChildElementCount() int {
	return countChildElements(e)
}

// Remove removes this element from its parent. No-op if it has no parent.
func (e *Element) Remove() {
	if e.parent != nil {
		e.parent.RemoveChild(e)
	}
}

// Document traversal methods

// Children returns all child Elements of this document.
func (d *Document) Children() []*Element {
	return collectChildElements(d)
}

// FirstElementChild returns the first child that is an Element, or nil.
func (d *Document) FirstElementChild() *Element {
	for c := d.firstChild; c != nil; c = c.NextSibling() {
		if el, ok := c.(*Element); ok {
			return el
		}
	}
	return nil
}

// LastElementChild returns the last child that is an Element, or nil.
func (d *Document) LastElementChild() *Element {
	for c := d.lastChild; c != nil; c = c.PreviousSibling() {
		if el, ok := c.(*Element); ok {
			return el
		}
	}
	return nil
}

// ChildElementCount returns the number of child Elements.
func (d *Document) ChildElementCount() int {
	return countChildElements(d)
}

// DocumentFragment traversal methods

// Children returns all child Elements of this fragment.
func (f *DocumentFragment) Children() []*Element {
	return collectChildElements(f)
}

// FirstElementChild returns the first child that is an Element, or nil.
func (f *DocumentFragment) FirstElementChild() *Element {
	for c := f.firstChild; c != nil; c = c.NextSibling() {
		if el, ok := c.(*Element); ok {
			return el
		}
	}
	return nil
}

// LastElementChild returns the last child that is an Element, or nil.
func (f *DocumentFragment) LastElementChild() *Element {
	for c := f.lastChild; c != nil; c = c.PreviousSibling() {
		if el, ok := c.(*Element); ok {
			return el
		}
	}
	return nil
}

// ChildElementCount returns the number of child Elements.
func (f *DocumentFragment) ChildElementCount() int {
	return countChildElements(f)
}

// Text ChildNode.Remove

// Remove removes this text node from its parent. No-op if it has no parent.
func (t *Text) Remove() {
	if t.parent != nil {
		t.parent.RemoveChild(t)
	}
}

// Comment ChildNode.Remove

// Remove removes this comment node from its parent. No-op if it has no parent.
func (c *Comment) Remove() {
	if c.parent != nil {
		c.parent.RemoveChild(c)
	}
}

// helpers

func collectChildElements(n Node) []*Element {
	var elements []*Element
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if el, ok := c.(*Element); ok {
			elements = append(elements, el)
		}
	}
	return elements
}

func countChildElements(n Node) int {
	count := 0
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(*Element); ok {
			count++
		}
	}
	return count
}
