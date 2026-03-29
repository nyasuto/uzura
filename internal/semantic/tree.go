// Package semantic provides structured page understanding for AI agents.
// It converts DOM trees into simplified semantic trees that expose
// landmarks, interactive elements, and content structure.
package semantic

import (
	"fmt"
	"strings"
)

// SemanticNode represents a node in the semantic tree.
// Each node captures the role, name, and optional value of a page element,
// along with a NodeID for referencing the corresponding DOM element.
type SemanticNode struct {
	Role     string          `json:"role"`               // "navigation", "main", "link", "button", "heading", "text", etc.
	Name     string          `json:"name"`               // Human-readable label (text content, aria-label, etc.)
	NodeID   int             `json:"node_id"`            // DOM element ID for interact tool reference
	Value    string          `json:"value,omitempty"`    // Input current value, link href, etc.
	Children []*SemanticNode `json:"children,omitempty"` // Child nodes
}

// Format returns a human-readable indented text representation of the tree.
// The depth parameter controls the initial indentation level.
func (n *SemanticNode) Format(depth int) string {
	var sb strings.Builder
	n.formatTo(&sb, depth)
	return sb.String()
}

func (n *SemanticNode) formatTo(sb *strings.Builder, depth int) {
	indent := strings.Repeat("  ", depth)
	sb.WriteString(indent)

	// Role with optional NodeID
	if n.NodeID > 0 && hasInteractableRole(n.Role) {
		fmt.Fprintf(sb, "[%s#%d]", n.Role, n.NodeID)
	} else {
		fmt.Fprintf(sb, "[%s]", n.Role)
	}

	// Name
	if n.Name != "" {
		sb.WriteString(" ")
		sb.WriteString(n.Name)
	}

	// Value (for links, inputs, etc.)
	if n.Value != "" {
		sb.WriteString(" → ")
		sb.WriteString(n.Value)
	}

	sb.WriteString("\n")

	for _, child := range n.Children {
		child.formatTo(sb, depth+1)
	}
}

// hasInteractableRole returns true for roles that can be targeted by the interact tool.
func hasInteractableRole(role string) bool {
	switch role {
	case "link", "button", "textbox", "checkbox", "radio", "combobox":
		return true
	default:
		return false
	}
}
