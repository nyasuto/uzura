package semantic

import (
	"strings"
	"testing"
)

func TestHeadingElements(t *testing.T) {
	html := `<html><body>
		<h1>Main Title</h1>
		<h2>Subtitle</h2>
		<h3>Section</h3>
	</body></html>`

	result := parseHTML(t, html)
	nodes := result.Nodes

	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}

	for _, n := range nodes {
		if n.Role != "heading" {
			t.Errorf("Role = %q, want %q", n.Role, "heading")
		}
	}
	if nodes[0].Name != "Main Title" {
		t.Errorf("Name = %q, want %q", nodes[0].Name, "Main Title")
	}
	if nodes[1].Name != "Subtitle" {
		t.Errorf("Name = %q, want %q", nodes[1].Name, "Subtitle")
	}
}

func TestHeadingTruncation(t *testing.T) {
	longTitle := strings.Repeat("A", 120)
	html := `<html><body><h1>` + longTitle + `</h1></body></html>`

	result := parseHTML(t, html)
	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if len(result.Nodes[0].Name) > 105 {
		t.Errorf("Name length = %d, should be truncated to ~103", len(result.Nodes[0].Name))
	}
}

func TestListElements(t *testing.T) {
	html := `<html><body>
		<ul>
			<li>First item</li>
			<li>Second item</li>
			<li>Third item</li>
		</ul>
	</body></html>`

	result := parseHTML(t, html)
	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	list := result.Nodes[0]
	if list.Role != "list" {
		t.Errorf("Role = %q, want %q", list.Role, "list")
	}
	if len(list.Children) != 3 {
		t.Fatalf("children = %d, want 3", len(list.Children))
	}
	for i, child := range list.Children {
		if child.Role != "listitem" {
			t.Errorf("child[%d].Role = %q, want %q", i, child.Role, "listitem")
		}
	}
	if list.Children[0].Name != "First item" {
		t.Errorf("child[0].Name = %q, want %q", list.Children[0].Name, "First item")
	}
}

func TestOrderedList(t *testing.T) {
	html := `<html><body>
		<ol>
			<li>Step 1</li>
			<li>Step 2</li>
		</ol>
	</body></html>`

	result := parseHTML(t, html)
	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Role != "list" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "list")
	}
}

func TestImageWithAlt(t *testing.T) {
	html := `<html><body>
		<img alt="Logo" src="logo.png">
		<img src="decorative.png">
	</body></html>`

	result := parseHTML(t, html)
	// Only the image with alt should appear
	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1 (decorative img without alt skipped)", len(result.Nodes))
	}
	if result.Nodes[0].Role != "image" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "image")
	}
	if result.Nodes[0].Name != "Logo" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Logo")
	}
}

func TestTableElement(t *testing.T) {
	html := `<html><body>
		<table>
			<tr><th>Name</th><th>Age</th><th>City</th></tr>
			<tr><td>Alice</td><td>30</td><td>Tokyo</td></tr>
			<tr><td>Bob</td><td>25</td><td>Osaka</td></tr>
		</table>
	</body></html>`

	result := parseHTML(t, html)
	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	tbl := result.Nodes[0]
	if tbl.Role != "table" {
		t.Errorf("Role = %q, want %q", tbl.Role, "table")
	}
	if tbl.Name != "3 rows × 3 cols" {
		t.Errorf("Name = %q, want %q", tbl.Name, "3 rows × 3 cols")
	}
}

func TestContentInLandmark(t *testing.T) {
	html := `<html><body>
		<main>
			<h1>Page Title</h1>
			<p>Some paragraph text</p>
			<ul>
				<li>Item 1</li>
				<li>Item 2</li>
			</ul>
			<img alt="Diagram" src="diagram.png">
		</main>
	</body></html>`

	result := parseHTML(t, html)
	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	main := result.Nodes[0]
	if main.Role != "main" {
		t.Errorf("Role = %q, want %q", main.Role, "main")
	}
	// Should have heading, list, and image (p is not a content element, text nodes skipped for now)
	if len(main.Children) != 3 {
		t.Fatalf("main children = %d, want 3", len(main.Children))
	}
	if main.Children[0].Role != "heading" {
		t.Errorf("child[0].Role = %q, want %q", main.Children[0].Role, "heading")
	}
	if main.Children[1].Role != "list" {
		t.Errorf("child[1].Role = %q, want %q", main.Children[1].Role, "list")
	}
	if main.Children[2].Role != "image" {
		t.Errorf("child[2].Role = %q, want %q", main.Children[2].Role, "image")
	}
}
