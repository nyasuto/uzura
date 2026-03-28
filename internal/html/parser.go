// Package html provides an HTML parser that builds a DOM tree.
package html

import (
	"io"
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() {
	dom.HTMLParseFragment = ParseFragment
}

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

// ParseFragment parses an HTML fragment in the context of a parent element
// and returns the resulting DOM nodes.
func ParseFragment(parent *dom.Element, htmlStr string) ([]dom.Node, error) {
	context := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Lookup([]byte(parent.LocalName())),
		Data:     parent.LocalName(),
	}
	nodes, err := html.ParseFragment(strings.NewReader(htmlStr), context)
	if err != nil {
		return nil, err
	}
	doc := parent.OwnerDocument()
	if doc == nil {
		doc = dom.NewDocument()
	}
	var result []dom.Node
	for _, n := range nodes {
		convertFragment(doc, &result, n)
	}
	return result, nil
}

func convertFragment(doc *dom.Document, result *[]dom.Node, n *html.Node) {
	switch n.Type {
	case html.ElementNode:
		elem := doc.CreateElement(n.Data)
		for _, attr := range n.Attr {
			elem.SetAttribute(attr.Key, attr.Val)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(doc, elem, c)
		}
		*result = append(*result, elem)
	case html.TextNode:
		*result = append(*result, doc.CreateTextNode(n.Data))
	case html.CommentNode:
		*result = append(*result, doc.CreateComment(n.Data))
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertFragment(doc, result, c)
		}
	}
}
