// Package html provides an HTML parser that builds a DOM tree.
package html

import (
	"io"

	"github.com/nyasuto/uzura/internal/dom"
	"golang.org/x/net/html"
)

// Parse reads HTML from r and returns a DOM Document.
func Parse(r io.Reader) (*dom.Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	doc := dom.NewDocument()
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		convertNode(doc, doc, c)
	}
	return doc, nil
}

func convertNode(doc *dom.Document, parent dom.Node, n *html.Node) {
	var node dom.Node

	switch n.Type {
	case html.ElementNode:
		elem := doc.CreateElement(n.Data)
		for _, attr := range n.Attr {
			elem.SetAttribute(attr.Key, attr.Val)
		}
		node = elem
	case html.TextNode:
		node = doc.CreateTextNode(n.Data)
	case html.CommentNode:
		node = doc.CreateComment(n.Data)
	case html.DocumentNode:
		// Skip the document node itself, process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(doc, parent, c)
		}
		return
	case html.DoctypeNode:
		// Skip doctype for now
		return
	default:
		return
	}

	parent.AppendChild(node)

	// Recursively convert children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		convertNode(doc, node, c)
	}
}
