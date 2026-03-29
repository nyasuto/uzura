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

	// Check for interactive elements
	if sn := b.processInteractive(elem); sn != nil {
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

// processInteractive checks if an element is an interactive element and returns
// a SemanticNode for it, or nil if it is not interactive.
func (b *Builder) processInteractive(elem *dom.Element) *SemanticNode {
	tag := elem.LocalName()

	switch tag {
	case "a":
		href := elem.GetAttribute("href")
		if href == "" {
			return nil
		}
		sn := b.makeNode("link", elem)
		sn.Value = href
		sn.Name = elementName(elem)
		return sn

	case "button":
		sn := b.makeNode("button", elem)
		sn.Name = elementName(elem)
		return sn

	case "input":
		return b.processInput(elem)

	case "select":
		sn := b.makeNode("combobox", elem)
		sn.Name = b.inputName(elem)
		sn.Value = selectedOptionText(elem)
		return sn

	case "textarea":
		sn := b.makeNode("textbox", elem)
		sn.Name = b.inputName(elem)
		return sn
	}

	return nil
}

// processInput handles <input> elements, mapping type to role.
func (b *Builder) processInput(elem *dom.Element) *SemanticNode {
	inputType := elem.GetAttribute("type")
	if inputType == "" {
		inputType = "text"
	}

	switch inputType {
	case "text", "email", "password", "search", "tel", "url", "number":
		sn := b.makeNode("textbox", elem)
		sn.Name = b.inputName(elem)
		return sn

	case "checkbox":
		sn := b.makeNode("checkbox", elem)
		sn.Name = b.inputName(elem)
		if elem.HasAttribute("checked") {
			sn.Value = "checked"
		} else {
			sn.Value = "unchecked"
		}
		return sn

	case "radio":
		sn := b.makeNode("radio", elem)
		sn.Name = b.inputName(elem)
		if elem.HasAttribute("checked") {
			sn.Value = "checked"
		} else {
			sn.Value = "unchecked"
		}
		return sn

	case "submit":
		sn := b.makeNode("button", elem)
		val := elem.GetAttribute("value")
		if val != "" {
			sn.Name = val
		} else {
			sn.Name = "Submit"
		}
		return sn

	case "hidden":
		return nil

	default:
		// Other input types treated as textbox
		sn := b.makeNode("textbox", elem)
		sn.Name = b.inputName(elem)
		return sn
	}
}

// inputName resolves the accessible name for an input element.
// Priority: aria-label > associated label (for attribute) > wrapping label > placeholder > name attribute.
func (b *Builder) inputName(elem *dom.Element) string {
	if label := elem.GetAttribute("aria-label"); label != "" {
		return label
	}

	// Check for associated <label for="id">
	if id := elem.GetAttribute("id"); id != "" {
		if doc := elem.OwnerDocument(); doc != nil {
			labels := doc.GetElementsByTagName("label")
			for _, labelEl := range labels {
				if labelEl.GetAttribute("for") == id {
					return labelEl.TextContent()
				}
			}
		}
	}

	// Check for wrapping <label>
	if labelText := findWrappingLabel(elem); labelText != "" {
		return labelText
	}

	if ph := elem.GetAttribute("placeholder"); ph != "" {
		return ph
	}

	if name := elem.GetAttribute("name"); name != "" {
		return name
	}

	return ""
}

// findWrappingLabel walks up the DOM tree to find a parent <label> element.
func findWrappingLabel(elem *dom.Element) string {
	for parent := elem.ParentNode(); parent != nil; parent = parent.ParentNode() {
		if pElem, ok := parent.(*dom.Element); ok {
			if pElem.LocalName() == "label" {
				return pElem.TextContent()
			}
		}
	}
	return ""
}

// selectedOptionText returns the text of the selected <option> in a <select>.
func selectedOptionText(selectElem *dom.Element) string {
	var firstOption string
	for child := selectElem.FirstChild(); child != nil; child = child.NextSibling() {
		opt, ok := child.(*dom.Element)
		if !ok || opt.LocalName() != "option" {
			continue
		}
		text := opt.TextContent()
		if firstOption == "" {
			firstOption = text
		}
		if opt.HasAttribute("selected") {
			return text
		}
	}
	return firstOption
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
