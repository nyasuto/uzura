package dom

import "strings"

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
	return d.appendChild(d, child)
}

// RemoveChild removes a child node from this document.
func (d *Document) RemoveChild(child Node) Node {
	return d.removeChild(child)
}

// InsertBefore inserts newChild before refChild.
func (d *Document) InsertBefore(newChild, refChild Node) Node {
	return d.insertBefore(d, newChild, refChild)
}

// TextContent for Document always returns empty string per spec.
func (d *Document) TextContent() string {
	return ""
}

// SetTextContent for Document is a no-op per spec.
func (d *Document) SetTextContent(text string) {}

// DocumentElement returns the root <html> element, or nil.
func (d *Document) DocumentElement() *Element {
	for c := d.firstChild; c != nil; c = c.NextSibling() {
		if e, ok := c.(*Element); ok && e.LocalName() == "html" {
			return e
		}
	}
	return nil
}

// Head returns the <head> element, or nil.
func (d *Document) Head() *Element {
	html := d.DocumentElement()
	if html == nil {
		return nil
	}
	return findChildElement(html, "head")
}

// Body returns the <body> element, or nil.
func (d *Document) Body() *Element {
	html := d.DocumentElement()
	if html == nil {
		return nil
	}
	return findChildElement(html, "body")
}

// Title returns the text content of the <title> element.
func (d *Document) Title() string {
	head := d.Head()
	if head == nil {
		return ""
	}
	title := findChildElement(head, "title")
	if title == nil {
		return ""
	}
	return title.TextContent()
}

// CreateElement creates a new Element with the given tag name, owned by this document.
func (d *Document) CreateElement(tagName string) *Element {
	e := NewElement(tagName)
	e.setOwnerDocument(d)
	return e
}

// CreateTextNode creates a new Text node owned by this document.
func (d *Document) CreateTextNode(data string) *Text {
	t := NewText(data)
	t.setOwnerDocument(d)
	return t
}

// CreateComment creates a new Comment node owned by this document.
func (d *Document) CreateComment(data string) *Comment {
	c := NewComment(data)
	c.setOwnerDocument(d)
	return c
}

// GetElementById returns the element with the given id, or nil.
func (d *Document) GetElementById(id string) *Element {
	return findElementById(d, id)
}

// GetElementsByTagName returns all elements with the given tag name.
func (d *Document) GetElementsByTagName(name string) []*Element {
	name = strings.ToLower(name)
	var result []*Element
	collectElementsByTagName(d, name, &result)
	return result
}

// GetElementsByClassName returns all elements that have all specified classes.
func (d *Document) GetElementsByClassName(classNames string) []*Element {
	classes := strings.Fields(classNames)
	if len(classes) == 0 {
		return nil
	}
	var result []*Element
	collectElementsByClassName(d, classes, &result)
	return result
}

// QuerySelector returns the first descendant element matching the CSS selector.
func (d *Document) QuerySelector(sel string) (*Element, error) {
	if SelectorQuery == nil {
		return nil, nil
	}
	return SelectorQuery(d, sel)
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
func (d *Document) QuerySelectorAll(sel string) ([]*Element, error) {
	if SelectorQueryAll == nil {
		return nil, nil
	}
	return SelectorQueryAll(d, sel)
}

func findChildElement(parent *Element, localName string) *Element {
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		if e, ok := c.(*Element); ok && e.LocalName() == localName {
			return e
		}
	}
	return nil
}

func findElementById(n Node, id string) *Element {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if e, ok := c.(*Element); ok {
			if e.Id() == id {
				return e
			}
			if found := findElementById(e, id); found != nil {
				return found
			}
		}
	}
	return nil
}

func collectElementsByTagName(n Node, name string, result *[]*Element) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if e, ok := c.(*Element); ok {
			if name == "*" || e.LocalName() == name {
				*result = append(*result, e)
			}
			collectElementsByTagName(e, name, result)
		}
	}
}

func collectElementsByClassName(n Node, classes []string, result *[]*Element) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if e, ok := c.(*Element); ok {
			if hasAllClasses(e, classes) {
				*result = append(*result, e)
			}
			collectElementsByClassName(e, classes, result)
		}
	}
}

func hasAllClasses(e *Element, classes []string) bool {
	elemClasses := strings.Fields(e.ClassName())
	for _, want := range classes {
		found := false
		for _, have := range elemClasses {
			if have == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
