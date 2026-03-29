package semantic

import (
	"fmt"
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
)

// processContent checks if an element is a content element (heading, list, image, table)
// and returns a SemanticNode for it, or nil if not a content element.
func (b *Builder) processContent(elem *dom.Element) *SemanticNode {
	tag := elem.LocalName()

	switch {
	case isHeading(tag):
		return b.processHeading(elem)
	case tag == "ul" || tag == "ol":
		return b.processList(elem)
	case tag == "img":
		return b.processImage(elem)
	case tag == "table":
		return b.processTable(elem)
	}

	return nil
}

// isHeading returns true for h1-h6 elements.
func isHeading(tag string) bool {
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

func (b *Builder) processHeading(elem *dom.Element) *SemanticNode {
	sn := b.makeNode("heading", elem)
	text := strings.TrimSpace(elem.TextContent())
	if len(text) > 100 {
		text = text[:100] + "…"
	}
	sn.Name = text
	return sn
}

func (b *Builder) processList(elem *dom.Element) *SemanticNode {
	sn := b.makeNode("list", elem)
	// Add list items as children
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		li, ok := child.(*dom.Element)
		if !ok || li.LocalName() != "li" {
			continue
		}
		liNode := &SemanticNode{
			Role: "listitem",
			Name: truncateText(li.TextContent(), 100),
		}
		sn.Children = append(sn.Children, liNode)
	}
	return sn
}

func (b *Builder) processImage(elem *dom.Element) *SemanticNode {
	alt := elem.GetAttribute("alt")
	if alt == "" {
		// Decorative image without alt text — skip
		return nil
	}
	sn := b.makeNode("image", elem)
	sn.Name = alt
	return sn
}

func (b *Builder) processTable(elem *dom.Element) *SemanticNode {
	rows, cols := tableSize(elem)
	sn := b.makeNode("table", elem)
	sn.Name = fmt.Sprintf("%d rows × %d cols", rows, cols)
	return sn
}

// tableSize counts rows and max columns in a table element.
func tableSize(table *dom.Element) (rows, cols int) {
	var maxCols int
	walkElements(table, func(el *dom.Element) {
		if el.LocalName() == "tr" {
			rows++
			colCount := 0
			for c := el.FirstChild(); c != nil; c = c.NextSibling() {
				if ce, ok := c.(*dom.Element); ok {
					tag := ce.LocalName()
					if tag == "td" || tag == "th" {
						colCount++
					}
				}
			}
			if colCount > maxCols {
				maxCols = colCount
			}
		}
	})
	return rows, maxCols
}

// walkElements recursively visits all element descendants.
func walkElements(parent dom.Node, fn func(*dom.Element)) {
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if el, ok := child.(*dom.Element); ok {
			fn(el)
			walkElements(el, fn)
		}
	}
}

// truncateText trims whitespace and truncates text to maxLen characters.
func truncateText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}
