package dom

// QueryEngine provides CSS selector matching for DOM trees.
// Implementations are injected into Document to avoid circular imports.
type QueryEngine interface {
	// QuerySelector returns the first descendant element matching the selector.
	QuerySelector(root Node, sel string) (*Element, error)
	// QuerySelectorAll returns all descendant elements matching the selector.
	QuerySelectorAll(root Node, sel string) ([]*Element, error)
	// Matches reports whether an element matches the selector.
	Matches(elem *Element, sel string) (bool, error)
	// Closest returns the closest ancestor (or self) matching the selector.
	Closest(elem *Element, sel string) (*Element, error)
}

// HTMLParseFragment parses an HTML fragment in the context of a parent element.
// It is set by the html package to avoid circular imports.
var HTMLParseFragment func(parent *Element, html string) ([]Node, error)

// getQueryEngine returns the QueryEngine from a node's owner document, or nil.
func getQueryEngine(n Node) QueryEngine {
	var doc *Document
	if d, ok := n.(*Document); ok {
		doc = d
	} else {
		doc = n.OwnerDocument()
	}
	if doc == nil {
		return nil
	}
	return doc.queryEngine
}
