// Package css provides CSS selector matching for DOM trees using cascadia.
package css

import (
	"github.com/andybalholm/cascadia"
	"github.com/nyasuto/uzura/internal/dom"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Engine implements dom.QueryEngine using the cascadia CSS selector library.
type Engine struct{}

// NewEngine returns a new CSS selector engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Ensure Engine implements dom.QueryEngine at compile time.
var _ dom.QueryEngine = (*Engine)(nil)

// QuerySelector returns the first descendant element matching the CSS selector, or nil.
func (e *Engine) QuerySelector(root dom.Node, sel string) (*dom.Element, error) {
	return QuerySelector(root, sel)
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
func (e *Engine) QuerySelectorAll(root dom.Node, sel string) ([]*dom.Element, error) {
	return QuerySelectorAll(root, sel)
}

// Matches reports whether the element matches the CSS selector.
func (e *Engine) Matches(elem *dom.Element, sel string) (bool, error) {
	return Matches(elem, sel)
}

// Closest returns the closest ancestor (or self) matching the CSS selector, or nil.
func (e *Engine) Closest(elem *dom.Element, sel string) (*dom.Element, error) {
	return Closest(elem, sel)
}

// Selector is a compiled CSS selector.
type Selector struct {
	sel cascadia.Selector
}

// Compile parses a CSS selector string and returns a compiled Selector.
func Compile(sel string) (*Selector, error) {
	compiled, err := cascadia.Compile(sel)
	if err != nil {
		return nil, err
	}
	return &Selector{sel: compiled}, nil
}

// QuerySelectorAll returns all descendant elements matching the CSS selector.
// The root node accepts either *dom.Document or *dom.Element.
func QuerySelectorAll(root dom.Node, sel string) ([]*dom.Element, error) {
	compiled, err := Compile(sel)
	if err != nil {
		return nil, err
	}
	return compiled.QueryAll(root), nil
}

// QuerySelector returns the first descendant element matching the CSS selector, or nil.
func QuerySelector(root dom.Node, sel string) (*dom.Element, error) {
	compiled, err := Compile(sel)
	if err != nil {
		return nil, err
	}
	return compiled.Query(root), nil
}

// QueryAll returns all descendant elements of root that match this selector.
func (s *Selector) QueryAll(root dom.Node) []*dom.Element {
	nodeMap := make(map[*html.Node]*dom.Element)
	htmlRoot := toHTMLNode(root, nodeMap)

	matches := s.sel.MatchAll(htmlRoot)
	var results []*dom.Element
	for _, m := range matches {
		if elem, ok := nodeMap[m]; ok {
			results = append(results, elem)
		}
	}
	return results
}

// Query returns the first descendant element of root that matches, or nil.
func (s *Selector) Query(root dom.Node) *dom.Element {
	nodeMap := make(map[*html.Node]*dom.Element)
	htmlRoot := toHTMLNode(root, nodeMap)

	match := s.sel.MatchFirst(htmlRoot)
	if match == nil {
		return nil
	}
	return nodeMap[match]
}

// Matches reports whether the element matches the CSS selector.
func Matches(elem *dom.Element, sel string) (bool, error) {
	compiled, err := cascadia.Compile(sel)
	if err != nil {
		return false, err
	}
	nodeMap := make(map[*html.Node]*dom.Element)
	// Build the full tree from the element's owner document or from the element
	// itself so that structural selectors (e.g. :first-child) work correctly.
	var root dom.Node = elem
	if doc := elem.OwnerDocument(); doc != nil {
		root = doc
	}
	htmlRoot := toHTMLNode(root, nodeMap)

	// Find the html.Node that corresponds to elem
	target := findHTMLNode(htmlRoot, nodeMap, elem)
	if target == nil {
		return false, nil
	}
	return compiled.Match(target), nil
}

// Closest returns the closest ancestor (or self) matching the CSS selector, or nil.
func Closest(elem *dom.Element, sel string) (*dom.Element, error) {
	compiled, err := cascadia.Compile(sel)
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[*html.Node]*dom.Element)
	var root dom.Node = elem
	if doc := elem.OwnerDocument(); doc != nil {
		root = doc
	}
	htmlRoot := toHTMLNode(root, nodeMap)

	target := findHTMLNode(htmlRoot, nodeMap, elem)
	if target == nil {
		return nil, nil
	}

	// Walk up from target checking each ancestor
	for n := target; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && compiled.Match(n) {
			return nodeMap[n], nil
		}
	}
	return nil, nil
}

// findHTMLNode finds the html.Node that maps to the given dom.Element.
func findHTMLNode(root *html.Node, nodeMap map[*html.Node]*dom.Element, target *dom.Element) *html.Node {
	for hn, elem := range nodeMap {
		if elem == target {
			return hn
		}
	}
	return nil
}

// toHTMLNode converts a dom.Node tree into an html.Node tree.
// Element nodes are recorded in nodeMap for reverse lookup.
func toHTMLNode(n dom.Node, nodeMap map[*html.Node]*dom.Element) *html.Node {
	hn := convertSingle(n, nodeMap)

	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		child := toHTMLNode(c, nodeMap)
		hn.AppendChild(child)
	}
	return hn
}

// convertSingle converts a single dom.Node (without children) to an html.Node.
func convertSingle(n dom.Node, nodeMap map[*html.Node]*dom.Element) *html.Node {
	switch v := n.(type) {
	case *dom.Document:
		return &html.Node{Type: html.DocumentNode}
	case *dom.Element:
		hn := &html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Lookup([]byte(v.LocalName())),
			Data:     v.LocalName(),
		}
		hn.Attr = append(hn.Attr, v.Attributes()...)
		nodeMap[hn] = v
		return hn
	case *dom.Text:
		return &html.Node{
			Type: html.TextNode,
			Data: v.TextContent(),
		}
	case *dom.Comment:
		return &html.Node{
			Type: html.CommentNode,
			Data: v.TextContent(),
		}
	default:
		return &html.Node{Type: html.TextNode}
	}
}
