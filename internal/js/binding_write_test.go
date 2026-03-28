package js

import (
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	_ "github.com/nyasuto/uzura/internal/html"
)

func TestDocumentCreateElement(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.createElement("div").tagName`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "DIV" {
		t.Errorf("createElement('div').tagName = %v, want 'DIV'", got)
	}
}

func TestDocumentCreateTextNode(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.createTextNode("hello").textContent`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("createTextNode('hello').textContent = %v, want 'hello'", got)
	}
}

func TestDocumentCreateDocumentFragment(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`typeof document.createDocumentFragment()`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "object" {
		t.Errorf("typeof createDocumentFragment() = %v, want 'object'", got)
	}
}

func TestElementSetAttribute(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`document.getElementById("content").setAttribute("data-new", "yes")`)
	if err != nil {
		t.Fatal(err)
	}

	el := doc.GetElementById("content")
	if v := el.GetAttribute("data-new"); v != "yes" {
		t.Errorf("Go side getAttribute('data-new') = %v, want 'yes'", v)
	}
}

func TestElementRemoveAttribute(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`document.getElementById("content").removeAttribute("data-value")`)
	if err != nil {
		t.Fatal(err)
	}

	el := doc.GetElementById("content")
	if el.HasAttribute("data-value") {
		t.Error("Go side: data-value should be removed")
	}
}

func TestElementTextContentSetter(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`document.getElementById("content").textContent = "New Text"`)
	if err != nil {
		t.Fatal(err)
	}

	el := doc.GetElementById("content")
	if got := el.TextContent(); got != "New Text" {
		t.Errorf("Go side textContent = %v, want 'New Text'", got)
	}
}

func TestElementInnerHTMLSetter(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`document.getElementById("content").innerHTML = "<b>Bold</b>"`)
	if err != nil {
		t.Fatal(err)
	}

	el := doc.GetElementById("content")
	inner := dom.InnerHTML(el)
	if inner != "<b>Bold</b>" {
		t.Errorf("Go side innerHTML = %v, want '<b>Bold</b>'", inner)
	}
}

func TestNodeAppendChild(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`
		var newEl = document.createElement("section");
		document.body.appendChild(newEl);
	`)
	if err != nil {
		t.Fatal(err)
	}

	body := doc.Body()
	last := body.LastChild()
	el, ok := last.(*dom.Element)
	if !ok {
		t.Fatalf("last child is %T, want *Element", last)
	}
	if el.TagName() != "SECTION" {
		t.Errorf("last child tagName = %v, want 'SECTION'", el.TagName())
	}
}

func TestNodeRemoveChild(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`
		var el = document.getElementById("content");
		el.parentNode.removeChild(el);
	`)
	if err != nil {
		t.Fatal(err)
	}

	if doc.GetElementById("content") != nil {
		t.Error("element should be removed from DOM")
	}
}

func TestNodeInsertBefore(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`
		var newEl = document.createElement("hr");
		var ref = document.querySelector("p");
		document.body.insertBefore(newEl, ref);
	`)
	if err != nil {
		t.Fatal(err)
	}

	body := doc.Body()
	children := body.ChildNodes()
	for i, child := range children {
		if el, ok := child.(*dom.Element); ok && el.TagName() == "HR" {
			if i+1 < len(children) {
				next, ok := children[i+1].(*dom.Element)
				if !ok || next.TagName() != "P" {
					t.Error("HR should be inserted before P")
				}
			}
			return
		}
	}
	t.Error("HR element not found in body children")
}

func TestNodeReplaceChild(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	_, err := vm.Eval(`
		var newEl = document.createElement("article");
		var oldEl = document.getElementById("content");
		oldEl.parentNode.replaceChild(newEl, oldEl);
	`)
	if err != nil {
		t.Fatal(err)
	}

	if doc.GetElementById("content") != nil {
		t.Error("old element should be removed")
	}
	elems := doc.GetElementsByTagName("article")
	if len(elems) != 1 {
		t.Errorf("article count = %d, want 1", len(elems))
	}
}

func TestElementClassListJS(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	// add
	_, err := vm.Eval(`document.getElementById("content").classList.add("new-class")`)
	if err != nil {
		t.Fatal(err)
	}
	el := doc.GetElementById("content")
	if !el.ClassList().Contains("new-class") {
		t.Error("classList.add failed")
	}

	// remove
	_, err = vm.Eval(`document.getElementById("content").classList.remove("active")`)
	if err != nil {
		t.Fatal(err)
	}
	if el.ClassList().Contains("active") {
		t.Error("classList.remove failed")
	}

	// toggle
	got, err := vm.Eval(`document.getElementById("content").classList.toggle("toggled")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Errorf("classList.toggle return = %v, want true", got)
	}

	// contains
	got, err = vm.Eval(`document.getElementById("content").classList.contains("container")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Errorf("classList.contains('container') = %v, want true", got)
	}
}

func TestElementDatasetJS(t *testing.T) {
	doc := makeTestDoc()
	vm := newTestVM(doc)

	// get
	got, err := vm.Eval(`document.getElementById("content").dataset.value`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("dataset.value = %v, want '42'", got)
	}

	// set
	_, err = vm.Eval(`document.getElementById("content").dataset.newKey = "hello"`)
	if err != nil {
		t.Fatal(err)
	}
	el := doc.GetElementById("content")
	if v := el.GetAttribute("data-new-key"); v != "hello" {
		t.Errorf("Go side data-new-key = %v, want 'hello'", v)
	}
}
