package semantic

import (
	"github.com/nyasuto/uzura/internal/dom"
)

// landmarkRoles maps HTML element tag names to their ARIA landmark roles.
var landmarkRoles = map[string]string{
	"header":  "banner",
	"nav":     "navigation",
	"main":    "main",
	"aside":   "complementary",
	"footer":  "contentinfo",
	"article": "article",
	"section": "region",
}

// Builder converts a DOM tree into a semantic tree.
type Builder struct {
	nextID int
	// NodeMap maps NodeIDs to DOM elements for interact tool reference.
	NodeMap map[int]*dom.Element
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{
		NodeMap: make(map[int]*dom.Element),
	}
}

// Build converts a DOM document into a semantic tree.
func (b *Builder) Build(doc *dom.Document) []*SemanticNode {
	var roots []*SemanticNode
	b.walkChildren(doc, &roots)
	return roots
}

func (b *Builder) walkChildren(parent dom.Node, out *[]*SemanticNode) {
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		b.processNode(child, out)
	}
}

func (b *Builder) processNode(n dom.Node, out *[]*SemanticNode) {
	elem, ok := n.(*dom.Element)
	if !ok {
		// Skip non-element nodes for now (text/comment handling in later tasks)
		return
	}

	tag := elem.LocalName()

	// Check for explicit ARIA role attribute first
	if ariaRole := elem.GetAttribute("role"); ariaRole != "" {
		sn := b.makeNode(ariaRole, elem)
		b.walkChildren(n, &sn.Children)
		*out = append(*out, sn)
		return
	}

	// Check for landmark elements
	if role, ok := landmarkRoles[tag]; ok {
		sn := b.makeNode(role, elem)
		b.walkChildren(n, &sn.Children)
		*out = append(*out, sn)
		return
	}

	// Non-landmark element: recurse into children, promoting them to current level
	b.walkChildren(n, out)
}

func (b *Builder) makeNode(role string, elem *dom.Element) *SemanticNode {
	b.nextID++
	id := b.nextID
	b.NodeMap[id] = elem

	name := elementName(elem)

	return &SemanticNode{
		Role:   role,
		Name:   name,
		NodeID: id,
	}
}

// elementName extracts a human-readable name for an element.
// It checks aria-label, aria-labelledby (TODO), and falls back to text content.
func elementName(elem *dom.Element) string {
	if label := elem.GetAttribute("aria-label"); label != "" {
		return label
	}
	// Fall back to trimmed text content, capped at 100 chars
	text := elem.TextContent()
	if len(text) > 100 {
		text = text[:100] + "…"
	}
	return text
}
