package dom

import "testing"

func TestDocumentFragment_BasicProperties(t *testing.T) {
	frag := NewDocumentFragment()

	if frag.NodeType() != DocumentFragmentNode {
		t.Errorf("NodeType = %d, want %d", frag.NodeType(), DocumentFragmentNode)
	}
	if frag.NodeName() != "#document-fragment" {
		t.Errorf("NodeName = %q, want %q", frag.NodeName(), "#document-fragment")
	}
	if frag.HasChildNodes() {
		t.Error("new fragment should have no children")
	}
}

func TestDocumentFragment_AppendChild(t *testing.T) {
	frag := NewDocumentFragment()
	a := NewElement("a")
	b := NewElement("b")

	frag.AppendChild(a)
	frag.AppendChild(b)

	if frag.FirstChild() != a {
		t.Error("FirstChild should be a")
	}
	if frag.LastChild() != b {
		t.Error("LastChild should be b")
	}
	if a.ParentNode() != frag {
		t.Error("a's parent should be frag")
	}
}

func TestDocumentFragment_AppendToElement_MovesChildren(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	frag := doc.CreateDocumentFragment()

	a := doc.CreateElement("span")
	b := doc.CreateTextNode("hello")
	c := doc.CreateElement("em")
	frag.AppendChild(a)
	frag.AppendChild(b)
	frag.AppendChild(c)

	parent.AppendChild(frag)

	// Fragment should be empty after insertion
	if frag.HasChildNodes() {
		t.Error("fragment should be empty after AppendChild")
	}

	// Parent should have the three children
	children := parent.ChildNodes()
	if len(children) != 3 {
		t.Fatalf("parent has %d children, want 3", len(children))
	}
	if children[0] != a {
		t.Error("first child should be a")
	}
	if children[1] != b {
		t.Error("second child should be b")
	}
	if children[2] != c {
		t.Error("third child should be c")
	}

	// Children should have parent set to the element, not the fragment
	if a.ParentNode() != parent {
		t.Error("a's parent should be parent element")
	}
}

func TestDocumentFragment_InsertBefore_MovesChildren(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	existing := doc.CreateElement("p")
	parent.AppendChild(existing)

	frag := doc.CreateDocumentFragment()
	a := doc.CreateElement("span")
	b := doc.CreateElement("em")
	frag.AppendChild(a)
	frag.AppendChild(b)

	parent.InsertBefore(frag, existing)

	children := parent.ChildNodes()
	if len(children) != 3 {
		t.Fatalf("parent has %d children, want 3", len(children))
	}
	if children[0] != a {
		t.Error("first child should be a (inserted before existing)")
	}
	if children[1] != b {
		t.Error("second child should be b (inserted before existing)")
	}
	if children[2] != existing {
		t.Error("third child should be existing")
	}

	if frag.HasChildNodes() {
		t.Error("fragment should be empty after InsertBefore")
	}
}

func TestDocumentFragment_ReplaceChild_MovesChildren(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	old := doc.CreateElement("p")
	tail := doc.CreateElement("footer")
	parent.AppendChild(old)
	parent.AppendChild(tail)

	frag := doc.CreateDocumentFragment()
	a := doc.CreateElement("span")
	b := doc.CreateElement("em")
	frag.AppendChild(a)
	frag.AppendChild(b)

	returned := parent.ReplaceChild(frag, old)
	if returned != old {
		t.Error("ReplaceChild should return the old child")
	}

	children := parent.ChildNodes()
	if len(children) != 3 {
		t.Fatalf("parent has %d children, want 3", len(children))
	}
	if children[0] != a {
		t.Error("first child should be a")
	}
	if children[1] != b {
		t.Error("second child should be b")
	}
	if children[2] != tail {
		t.Error("third child should be tail")
	}
}

func TestDocumentFragment_ReplaceChild_LastChild(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	head := doc.CreateElement("header")
	old := doc.CreateElement("p")
	parent.AppendChild(head)
	parent.AppendChild(old)

	frag := doc.CreateDocumentFragment()
	a := doc.CreateElement("span")
	frag.AppendChild(a)

	parent.ReplaceChild(frag, old)

	children := parent.ChildNodes()
	if len(children) != 2 {
		t.Fatalf("parent has %d children, want 2", len(children))
	}
	if children[0] != head {
		t.Error("first child should be head")
	}
	if children[1] != a {
		t.Error("second child should be a")
	}
	if parent.LastChild() != a {
		t.Error("lastChild should be a")
	}
}

func TestDocumentFragment_EmptyFragment_AppendChild(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div")
	existing := doc.CreateElement("p")
	parent.AppendChild(existing)

	frag := doc.CreateDocumentFragment()
	parent.AppendChild(frag)

	children := parent.ChildNodes()
	if len(children) != 1 {
		t.Fatalf("parent has %d children, want 1", len(children))
	}
}

func TestDocumentFragment_TextContent(t *testing.T) {
	frag := NewDocumentFragment()
	a := NewElement("span")
	a.AppendChild(NewText("hello"))
	b := NewText(" world")
	frag.AppendChild(a)
	frag.AppendChild(b)

	if got := frag.TextContent(); got != "hello world" {
		t.Errorf("TextContent = %q, want %q", got, "hello world")
	}
}

func TestDocumentFragment_SetTextContent(t *testing.T) {
	frag := NewDocumentFragment()
	frag.AppendChild(NewElement("div"))
	frag.AppendChild(NewElement("span"))

	frag.SetTextContent("replaced")

	children := frag.ChildNodes()
	if len(children) != 1 {
		t.Fatalf("children count = %d, want 1", len(children))
	}
	text, ok := children[0].(*Text)
	if !ok {
		t.Fatal("child should be a Text node")
	}
	if text.Data != "replaced" {
		t.Errorf("text data = %q, want %q", text.Data, "replaced")
	}
}

func TestDocumentFragment_CloneNode(t *testing.T) {
	frag := NewDocumentFragment()
	a := NewElement("div")
	a.SetAttribute("id", "x")
	b := NewText("hello")
	frag.AppendChild(a)
	frag.AppendChild(b)

	// Shallow clone
	shallow := frag.CloneNode(false)
	sf, ok := shallow.(*DocumentFragment)
	if !ok {
		t.Fatal("shallow clone should be *DocumentFragment")
	}
	if sf.HasChildNodes() {
		t.Error("shallow clone should have no children")
	}

	// Deep clone
	deep := frag.CloneNode(true)
	df, ok := deep.(*DocumentFragment)
	if !ok {
		t.Fatal("deep clone should be *DocumentFragment")
	}
	children := df.ChildNodes()
	if len(children) != 2 {
		t.Fatalf("deep clone has %d children, want 2", len(children))
	}
	if children[0].(*Element).GetAttribute("id") != "x" {
		t.Error("deep clone should preserve attributes")
	}
	// Verify it's a separate copy
	if children[0] == a {
		t.Error("deep clone children should be different objects")
	}
}

func TestDocumentFragment_IsEqualNode(t *testing.T) {
	f1 := NewDocumentFragment()
	f1.AppendChild(NewText("hello"))

	f2 := NewDocumentFragment()
	f2.AppendChild(NewText("hello"))

	if !f1.IsEqualNode(f2) {
		t.Error("fragments with same content should be equal")
	}

	f2.AppendChild(NewText(" world"))
	if f1.IsEqualNode(f2) {
		t.Error("fragments with different content should not be equal")
	}
}

func TestDocumentFragment_Serialize(t *testing.T) {
	frag := NewDocumentFragment()
	frag.AppendChild(NewElement("div"))
	frag.AppendChild(NewText("hello"))

	got := Serialize(frag)
	want := "<div></div>hello"
	if got != want {
		t.Errorf("Serialize = %q, want %q", got, want)
	}
}

func TestCreateDocumentFragment(t *testing.T) {
	doc := NewDocument()
	frag := doc.CreateDocumentFragment()

	if frag.NodeType() != DocumentFragmentNode {
		t.Error("wrong node type")
	}
	if frag.OwnerDocument() != doc {
		t.Error("ownerDocument should be the creating document")
	}
}

func TestImportNode_Shallow(t *testing.T) {
	doc1 := NewDocument()
	doc2 := NewDocument()

	elem := doc1.CreateElement("div")
	elem.SetAttribute("class", "test")
	elem.AppendChild(doc1.CreateTextNode("child"))

	imported := doc2.ImportNode(elem, false)
	ie, ok := imported.(*Element)
	if !ok {
		t.Fatal("imported node should be *Element")
	}

	if ie.OwnerDocument() != doc2 {
		t.Error("imported node should belong to doc2")
	}
	if ie.GetAttribute("class") != "test" {
		t.Error("imported node should preserve attributes")
	}
	if ie.HasChildNodes() {
		t.Error("shallow import should not include children")
	}
	// Original should be unchanged
	if elem.OwnerDocument() != doc1 {
		t.Error("original should still belong to doc1")
	}
	if !elem.HasChildNodes() {
		t.Error("original should still have children")
	}
}

func TestImportNode_Deep(t *testing.T) {
	doc1 := NewDocument()
	doc2 := NewDocument()

	elem := doc1.CreateElement("div")
	child := doc1.CreateElement("span")
	child.SetAttribute("id", "inner")
	text := doc1.CreateTextNode("hello")
	child.AppendChild(text)
	elem.AppendChild(child)

	imported := doc2.ImportNode(elem, true)
	ie := imported.(*Element)

	if ie.OwnerDocument() != doc2 {
		t.Error("imported root should belong to doc2")
	}

	children := ie.ChildNodes()
	if len(children) != 1 {
		t.Fatalf("imported has %d children, want 1", len(children))
	}

	importedChild := children[0].(*Element)
	if importedChild.OwnerDocument() != doc2 {
		t.Error("imported child should belong to doc2")
	}
	if importedChild.GetAttribute("id") != "inner" {
		t.Error("imported child should preserve attributes")
	}

	grandchildren := importedChild.ChildNodes()
	if len(grandchildren) != 1 {
		t.Fatal("imported grandchild missing")
	}
	if grandchildren[0].OwnerDocument() != doc2 {
		t.Error("imported grandchild should belong to doc2")
	}
}

func TestImportNode_DocumentFragment(t *testing.T) {
	doc1 := NewDocument()
	doc2 := NewDocument()

	frag := doc1.CreateDocumentFragment()
	frag.AppendChild(doc1.CreateElement("a"))
	frag.AppendChild(doc1.CreateElement("b"))

	imported := doc2.ImportNode(frag, true)
	ifrag, ok := imported.(*DocumentFragment)
	if !ok {
		t.Fatal("imported should be *DocumentFragment")
	}

	if ifrag.OwnerDocument() != doc2 {
		t.Error("imported fragment should belong to doc2")
	}

	children := ifrag.ChildNodes()
	if len(children) != 2 {
		t.Fatalf("imported fragment has %d children, want 2", len(children))
	}
	for _, c := range children {
		if c.OwnerDocument() != doc2 {
			t.Error("imported fragment child should belong to doc2")
		}
	}

	// Original unchanged
	if len(frag.ChildNodes()) != 2 {
		t.Error("original fragment should be unchanged")
	}
}
