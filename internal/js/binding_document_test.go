package js

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/css"
	"github.com/nyasuto/uzura/internal/dom"
)

func makeTestDoc() *dom.Document {
	doc := dom.NewDocument()
	qe := css.NewEngine()
	doc.SetQueryEngine(qe)

	html := doc.CreateElement("html")
	head := doc.CreateElement("head")
	title := doc.CreateElement("title")
	title.AppendChild(doc.CreateTextNode("Test Page"))
	head.AppendChild(title)
	html.AppendChild(head)

	body := doc.CreateElement("body")
	body.SetAttribute("class", "main")

	div := doc.CreateElement("div")
	div.SetAttribute("id", "content")
	div.SetAttribute("class", "container active")
	div.SetAttribute("data-value", "42")
	div.AppendChild(doc.CreateTextNode("Hello World"))
	body.AppendChild(div)

	p := doc.CreateElement("p")
	p.SetAttribute("class", "text")
	p.AppendChild(doc.CreateTextNode("Paragraph"))
	body.AppendChild(p)

	span := doc.CreateElement("span")
	span.SetAttribute("class", "text highlight")
	span.AppendChild(doc.CreateTextNode("Span"))
	body.AppendChild(span)

	html.AppendChild(body)
	doc.AppendChild(html)
	return doc
}

func newTestVM(doc *dom.Document) *VM {
	vm := New()
	BindDocument(vm, doc)
	return vm
}

func TestDocumentTitle(t *testing.T) {
	vm := newTestVM(makeTestDoc())
	got, err := vm.Eval(`document.title`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Test Page" {
		t.Errorf("document.title = %v, want 'Test Page'", got)
	}
}

func TestDocumentDocumentElement(t *testing.T) {
	vm := newTestVM(makeTestDoc())
	got, err := vm.Eval(`document.documentElement.tagName`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "HTML" {
		t.Errorf("document.documentElement.tagName = %v, want 'HTML'", got)
	}
}

func TestDocumentHead(t *testing.T) {
	vm := newTestVM(makeTestDoc())
	got, err := vm.Eval(`document.head.tagName`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "HEAD" {
		t.Errorf("document.head.tagName = %v, want 'HEAD'", got)
	}
}

func TestDocumentBody(t *testing.T) {
	vm := newTestVM(makeTestDoc())
	got, err := vm.Eval(`document.body.tagName`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "BODY" {
		t.Errorf("document.body.tagName = %v, want 'BODY'", got)
	}
}

func TestDocumentGetElementById(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementById("content").textContent`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hello World" {
		t.Errorf("getElementById result = %v, want 'Hello World'", got)
	}

	got, err = vm.Eval(`document.getElementById("nonexistent")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("getElementById('nonexistent') = %v, want nil", got)
	}
}

func TestDocumentQuerySelector(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.querySelector(".container").id`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "content" {
		t.Errorf("querySelector result id = %v, want 'content'", got)
	}
}

func TestDocumentQuerySelectorAll(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.querySelectorAll(".text").length`)
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(2) {
		t.Errorf("querySelectorAll('.text').length = %v, want 2", got)
	}
}

func TestDocumentGetElementsByTagName(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementsByTagName("div").length`)
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(1) {
		t.Errorf("getElementsByTagName('div').length = %v, want 1", got)
	}
}

func TestDocumentGetElementsByClassName(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementsByClassName("text").length`)
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(2) {
		t.Errorf("getElementsByClassName('text').length = %v, want 2", got)
	}
}

func TestElementProperties(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	tests := []struct {
		name   string
		script string
		want   interface{}
	}{
		{"tagName", `document.getElementById("content").tagName`, "DIV"},
		{"id", `document.getElementById("content").id`, "content"},
		{"className", `document.getElementById("content").className`, "container active"},
		{"textContent", `document.getElementById("content").textContent`, "Hello World"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vm.Eval(tt.script)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tt.want {
				t.Errorf("%s = %v, want %v", tt.script, got, tt.want)
			}
		})
	}
}

func TestElementGetAttribute(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementById("content").getAttribute("data-value")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("getAttribute('data-value') = %v, want '42'", got)
	}

	got, err = vm.Eval(`document.getElementById("content").getAttribute("nonexist")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("getAttribute('nonexist') = %v, want nil", got)
	}
}

func TestElementHasAttribute(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementById("content").hasAttribute("id")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Errorf("hasAttribute('id') = %v, want true", got)
	}

	got, err = vm.Eval(`document.getElementById("content").hasAttribute("nope")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != false {
		t.Errorf("hasAttribute('nope') = %v, want false", got)
	}
}

func TestElementQuerySelector(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.body.querySelector("p").textContent`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Paragraph" {
		t.Errorf("body.querySelector('p').textContent = %v, want 'Paragraph'", got)
	}
}

func TestElementQuerySelectorAll(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.body.querySelectorAll(".text").length`)
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(2) {
		t.Errorf("body.querySelectorAll('.text').length = %v, want 2", got)
	}
}

func TestElementMatches(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementById("content").matches(".container")`)
	if err != nil {
		t.Fatal(err)
	}
	if got != true {
		t.Errorf("matches('.container') = %v, want true", got)
	}
}

func TestElementClosest(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementById("content").closest("body").tagName`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "BODY" {
		t.Errorf("closest('body').tagName = %v, want 'BODY'", got)
	}
}

func TestNodeListForEach(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`
		var result = [];
		document.querySelectorAll(".text").forEach(function(el) {
			result.push(el.tagName);
		});
		result.join(",");
	`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "P,SPAN" {
		t.Errorf("forEach result = %v, want 'P,SPAN'", got)
	}
}

func TestNodeListIndexAccess(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.querySelectorAll(".text")[0].tagName`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "P" {
		t.Errorf("[0].tagName = %v, want 'P'", got)
	}
}

func TestElementInnerHTML(t *testing.T) {
	vm := newTestVM(makeTestDoc())

	got, err := vm.Eval(`document.getElementById("content").innerHTML`)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := got.(string)
	if !ok {
		t.Fatalf("innerHTML is not string: %T", got)
	}
	if !strings.Contains(s, "Hello World") {
		t.Errorf("innerHTML = %v, want to contain 'Hello World'", s)
	}
}
