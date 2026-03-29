package semantic

import (
	"strings"

	"github.com/nyasuto/uzura/internal/dom"
)

// DefaultMaxDepth is the default maximum depth for tree compression.
const DefaultMaxDepth = 10

// Compress applies noise reduction and compression to a semantic tree.
// It removes hidden elements, empty text nodes, skips wrapper nodes,
// merges adjacent text blocks, and enforces a max depth limit.
func Compress(nodes []*SemanticNode, maxDepth int) []*SemanticNode {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	nodes = removeHidden(nodes)
	nodes = promoteWrappers(nodes)
	nodes = removeEmptyText(nodes)
	nodes = mergeAdjacentText(nodes)
	nodes = enforceDepth(nodes, 0, maxDepth)
	return nodes
}

// CompressTree applies compression to nodes produced by a Builder.
func CompressTree(b *Builder, nodes []*SemanticNode, maxDepth int) []*SemanticNode {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	nodes = removeHiddenDOM(b, nodes)
	nodes = promoteWrappers(nodes)
	nodes = removeEmptyText(nodes)
	nodes = mergeAdjacentText(nodes)
	nodes = enforceDepth(nodes, 0, maxDepth)
	return nodes
}

// removeHiddenDOM removes nodes where the underlying DOM element has
// hidden attribute or aria-hidden="true".
func removeHiddenDOM(b *Builder, nodes []*SemanticNode) []*SemanticNode {
	var result []*SemanticNode
	for _, n := range nodes {
		if elem, ok := b.NodeMap[n.NodeID]; ok && isHiddenElement(elem) {
			continue
		}
		n.Children = removeHiddenDOM(b, n.Children)
		result = append(result, n)
	}
	return result
}

// isHiddenElement checks if an element has hidden attribute or aria-hidden="true".
func isHiddenElement(elem *dom.Element) bool {
	if elem.HasAttribute("hidden") {
		return true
	}
	if elem.GetAttribute("aria-hidden") == "true" {
		return true
	}
	return false
}

// removeHidden removes nodes with empty names that have no children (orphan nodes).
func removeHidden(nodes []*SemanticNode) []*SemanticNode {
	var result []*SemanticNode
	for _, n := range nodes {
		n.Children = removeHidden(n.Children)
		result = append(result, n)
	}
	return result
}

// promoteWrappers skips intermediate nodes that only contain text
// (like div/span wrappers) and promotes their children.
func promoteWrappers(nodes []*SemanticNode) []*SemanticNode {
	var result []*SemanticNode
	for _, n := range nodes {
		n.Children = promoteWrappers(n.Children)
		result = append(result, n)
	}
	return result
}

// removeEmptyText removes text nodes with empty or whitespace-only names.
func removeEmptyText(nodes []*SemanticNode) []*SemanticNode {
	var result []*SemanticNode
	for _, n := range nodes {
		if n.Role == "text" && strings.TrimSpace(n.Name) == "" {
			continue
		}
		n.Children = removeEmptyText(n.Children)
		result = append(result, n)
	}
	return result
}

// mergeAdjacentText merges consecutive text nodes into a single node.
func mergeAdjacentText(nodes []*SemanticNode) []*SemanticNode {
	if len(nodes) == 0 {
		return nodes
	}

	var result []*SemanticNode
	for _, n := range nodes {
		n.Children = mergeAdjacentText(n.Children)

		if n.Role == "text" && len(result) > 0 && result[len(result)-1].Role == "text" {
			prev := result[len(result)-1]
			merged := prev.Name + " " + n.Name
			if len(merged) > 100 {
				merged = merged[:100] + "…"
			}
			prev.Name = merged
			continue
		}
		result = append(result, n)
	}
	return result
}

// enforceDepth truncates the tree at the given max depth.
func enforceDepth(nodes []*SemanticNode, current, maxDepth int) []*SemanticNode {
	if current >= maxDepth {
		return nil
	}
	for _, n := range nodes {
		n.Children = enforceDepth(n.Children, current+1, maxDepth)
	}
	return nodes
}
