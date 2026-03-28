package dom

import (
	"testing"
)

func TestAppendChild(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (parent *Element, child Node)
		wantFirst string
		wantLast  string
		wantCount int
	}{
		{
			name: "append single child",
			setup: func() (*Element, Node) {
				p := NewElement("div")
				c := NewElement("span")
				return p, c
			},
			wantFirst: "SPAN",
			wantLast:  "SPAN",
			wantCount: 1,
		},
		{
			name: "append multiple children",
			setup: func() (*Element, Node) {
				p := NewElement("div")
				p.AppendChild(NewElement("a"))
				c := NewElement("b")
				return p, c
			},
			wantFirst: "A",
			wantLast:  "B",
			wantCount: 2,
		},
		{
			name: "append text node",
			setup: func() (*Element, Node) {
				p := NewElement("p")
				c := NewText("hello")
				return p, c
			},
			wantFirst: "#text",
			wantLast:  "#text",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, child := tt.setup()
			parent.AppendChild(child)

			if got := parent.FirstChild().NodeName(); got != tt.wantFirst {
				t.Errorf("FirstChild.NodeName() = %q, want %q", got, tt.wantFirst)
			}
			if got := parent.LastChild().NodeName(); got != tt.wantLast {
				t.Errorf("LastChild.NodeName() = %q, want %q", got, tt.wantLast)
			}
			if got := len(parent.ChildNodes()); got != tt.wantCount {
				t.Errorf("len(ChildNodes()) = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestAppendChildDetachesFromOldParent(t *testing.T) {
	oldParent := NewElement("div")
	newParent := NewElement("section")
	child := NewElement("span")

	oldParent.AppendChild(child)
	if child.ParentNode() != oldParent {
		t.Fatal("child should be attached to oldParent")
	}

	newParent.AppendChild(child)
	if child.ParentNode() != newParent {
		t.Error("child should be attached to newParent")
	}
	if len(oldParent.ChildNodes()) != 0 {
		t.Error("oldParent should have no children")
	}
}

func TestRemoveChild(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() (*Element, Node)
		wantRemaining int
	}{
		{
			name: "remove only child",
			setup: func() (*Element, Node) {
				p := NewElement("div")
				c := NewElement("span")
				p.AppendChild(c)
				return p, c
			},
			wantRemaining: 0,
		},
		{
			name: "remove first child",
			setup: func() (*Element, Node) {
				p := NewElement("div")
				c1 := NewElement("a")
				p.AppendChild(c1)
				p.AppendChild(NewElement("b"))
				return p, c1
			},
			wantRemaining: 1,
		},
		{
			name: "remove last child",
			setup: func() (*Element, Node) {
				p := NewElement("div")
				p.AppendChild(NewElement("a"))
				c2 := NewElement("b")
				p.AppendChild(c2)
				return p, c2
			},
			wantRemaining: 1,
		},
		{
			name: "remove middle child",
			setup: func() (*Element, Node) {
				p := NewElement("div")
				p.AppendChild(NewElement("a"))
				c2 := NewElement("b")
				p.AppendChild(c2)
				p.AppendChild(NewElement("c"))
				return p, c2
			},
			wantRemaining: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, child := tt.setup()
			removed := parent.RemoveChild(child)

			if removed != child {
				t.Error("RemoveChild should return the removed node")
			}
			if child.ParentNode() != nil {
				t.Error("removed child should have nil parent")
			}
			if child.PreviousSibling() != nil {
				t.Error("removed child should have nil prev sibling")
			}
			if child.NextSibling() != nil {
				t.Error("removed child should have nil next sibling")
			}
			if got := len(parent.ChildNodes()); got != tt.wantRemaining {
				t.Errorf("remaining children = %d, want %d", got, tt.wantRemaining)
			}
		})
	}
}

func TestRemoveChildSiblingPointers(t *testing.T) {
	p := NewElement("div")
	a := NewElement("a")
	b := NewElement("b")
	c := NewElement("c")
	p.AppendChild(a)
	p.AppendChild(b)
	p.AppendChild(c)

	p.RemoveChild(b)

	if a.NextSibling() != c {
		t.Error("a.NextSibling should be c after removing b")
	}
	if c.PreviousSibling() != a {
		t.Error("c.PreviousSibling should be a after removing b")
	}
}

func TestInsertBefore(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*Element, Node, Node)
		wantOrder []string
	}{
		{
			name: "insert before first child",
			setup: func() (*Element, Node, Node) {
				p := NewElement("div")
				a := NewElement("a")
				p.AppendChild(a)
				n := NewElement("new")
				return p, n, a
			},
			wantOrder: []string{"NEW", "A"},
		},
		{
			name: "insert before last child",
			setup: func() (*Element, Node, Node) {
				p := NewElement("div")
				a := NewElement("a")
				b := NewElement("b")
				p.AppendChild(a)
				p.AppendChild(b)
				n := NewElement("new")
				return p, n, b
			},
			wantOrder: []string{"A", "NEW", "B"},
		},
		{
			name: "insert before nil appends",
			setup: func() (*Element, Node, Node) {
				p := NewElement("div")
				a := NewElement("a")
				p.AppendChild(a)
				n := NewElement("new")
				return p, n, nil
			},
			wantOrder: []string{"A", "NEW"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, newChild, refChild := tt.setup()
			parent.InsertBefore(newChild, refChild)

			children := parent.ChildNodes()
			if len(children) != len(tt.wantOrder) {
				t.Fatalf("children count = %d, want %d", len(children), len(tt.wantOrder))
			}
			for i, want := range tt.wantOrder {
				if got := children[i].NodeName(); got != want {
					t.Errorf("child[%d].NodeName() = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestInsertBeforeDetachesFromOldParent(t *testing.T) {
	oldParent := NewElement("div")
	newParent := NewElement("section")
	ref := NewElement("ref")
	newParent.AppendChild(ref)

	child := NewElement("child")
	oldParent.AppendChild(child)

	newParent.InsertBefore(child, ref)

	if len(oldParent.ChildNodes()) != 0 {
		t.Error("oldParent should have no children")
	}
	if child.ParentNode() != newParent {
		t.Error("child should be attached to newParent")
	}
}
