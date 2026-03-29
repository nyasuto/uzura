package dom

import "strings"

// Document represents a DOM Document node.
type Document struct {
	baseNode
	queryEngine QueryEngine
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

// ReplaceChild replaces oldChild with newChild. Returns oldChild.
func (d *Document) ReplaceChild(newChild, oldChild Node) Node {
	return d.replaceChild(d, newChild, oldChild)
}

// CloneNode returns a copy of this document. If deep is true, all descendants are also cloned.
func (d *Document) CloneNode(deep bool) Node {
	clone := NewDocument()
	if deep {
		for c := d.firstChild; c != nil; c = c.NextSibling() {
			child := c.CloneNode(true)
			clone.AppendChild(child)
		}
	}
	return clone
}

// Contains reports whether other is a descendant of this document (or is itself).
func (d *Document) Contains(other Node) bool {
	return d.contains(d, other)
}

// HasChildNodes reports whether this document has any children.
func (d *Document) HasChildNodes() bool {
	return d.hasChildNodes()
}

// Normalize merges adjacent Text nodes and removes empty Text nodes.
func (d *Document) Normalize() {
	d.normalize(d)
}

// IsEqualNode reports whether other is structurally equal to this document.
func (d *Document) IsEqualNode(other Node) bool {
	return isEqualNode(d, other)
}

// IsSameNode reports whether other is the exact same node reference.
func (d *Document) IsSameNode(other Node) bool {
	return d.isSameNode(d, other)
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

// SetTitle sets the text content of the <title> element, creating one if needed.
func (d *Document) SetTitle(title string) {
	head := d.Head()
	if head == nil {
		return
	}
	titleEl := findChildElement(head, "title")
	if titleEl == nil {
		titleEl = d.CreateElement("title")
		head.AppendChild(titleEl)
	}
	titleEl.SetTextContent(title)
}

// CreateDocumentFragment creates a new empty DocumentFragment owned by this document.
func (d *Document) CreateDocumentFragment() *DocumentFragment {
	f := NewDocumentFragment()
	f.setOwnerDocument(d)
	return f
}

// ImportNode returns a copy of a node from another document, suitable for insertion.
// If deep is true, the entire subtree is cloned. The ownerDocument of the result
// is set to this document.
func (d *Document) ImportNode(node Node, deep bool) Node {
	clone := node.CloneNode(deep)
	setOwnerDocumentRecursive(clone, d)
	return clone
}

// SetQueryEngine sets the CSS selector engine for this document.
func (d *Document) SetQueryEngine(qe QueryEngine) {
	d.queryEngine = qe
}

// GetQueryEngine returns the CSS selector engine for this document, or nil.
func (d *Document) GetQueryEngine() QueryEngine {
	return d.queryEngine
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
	if d.queryEngine == nil {
		return nil, nil
	}
	return d.queryEngine.QuerySelector(d, sel)
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
func (d *Document) QuerySelectorAll(sel string) ([]*Element, error) {
	if d.queryEngine == nil {
		return nil, nil
	}
	return d.queryEngine.QuerySelectorAll(d, sel)
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

func setOwnerDocumentRecursive(n Node, doc *Document) {
	n.setOwnerDocument(doc)
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		setOwnerDocumentRecursive(c, doc)
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
