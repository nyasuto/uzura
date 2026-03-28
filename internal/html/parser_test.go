package html

import (
	"strings"
	"testing"
)

func TestParseBasicHTML(t *testing.T) {
	input := `<html><head><title>Test</title></head><body><p>Hello</p></body></html>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if doc.DocumentElement() == nil {
		t.Fatal("DocumentElement should not be nil")
	}
	if doc.Head() == nil {
		t.Fatal("Head should not be nil")
	}
	if doc.Body() == nil {
		t.Fatal("Body should not be nil")
	}
	if got := doc.Title(); got != "Test" {
		t.Errorf("Title() = %q, want %q", got, "Test")
	}
}

func TestParseImplicitTags(t *testing.T) {
	// html parser should insert missing html/head/body
	input := `<p>Hello</p>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if doc.DocumentElement() == nil {
		t.Fatal("html element should be implicitly created")
	}
	if doc.Body() == nil {
		t.Fatal("body should be implicitly created")
	}

	// The <p> should be inside <body>
	body := doc.Body()
	children := body.ChildNodes()
	found := false
	for _, c := range children {
		if c.NodeName() == "P" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected <p> inside <body>")
	}
}

func TestParseNestedTable(t *testing.T) {
	input := `<table><tr><td>Cell 1</td><td>Cell 2</td></tr></table>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	tables := doc.GetElementsByTagName("table")
	if len(tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(tables))
	}
	tds := doc.GetElementsByTagName("td")
	if len(tds) != 2 {
		t.Errorf("expected 2 td, got %d", len(tds))
	}
}

func TestParseMissingCloseTags(t *testing.T) {
	input := `<div><p>Para 1<p>Para 2</div>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	ps := doc.GetElementsByTagName("p")
	if len(ps) != 2 {
		t.Errorf("expected 2 <p>, got %d", len(ps))
	}
}

func TestParseEmptyDocument(t *testing.T) {
	doc, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Even empty input should produce html/head/body
	if doc.DocumentElement() == nil {
		t.Error("DocumentElement should exist even for empty input")
	}
}

func TestParseWithAttributes(t *testing.T) {
	input := `<div id="main" class="container"><a href="https://example.com">Link</a></div>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	div := doc.GetElementById("main")
	if div == nil {
		t.Fatal("should find div with id=main")
	}
	if div.ClassName() != "container" {
		t.Errorf("ClassName() = %q, want %q", div.ClassName(), "container")
	}

	links := doc.GetElementsByTagName("a")
	if len(links) != 1 {
		t.Fatalf("expected 1 <a>, got %d", len(links))
	}
	if got := links[0].GetAttribute("href"); got != "https://example.com" {
		t.Errorf("href = %q, want %q", got, "https://example.com")
	}
}

func TestParseComment(t *testing.T) {
	input := `<html><body><!-- a comment --><p>text</p></body></html>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	body := doc.Body()
	if body == nil {
		t.Fatal("Body should not be nil")
	}

	// First child of body should be the comment
	first := body.FirstChild()
	if first == nil {
		t.Fatal("body should have children")
	}
	if first.NodeName() != "#comment" {
		t.Errorf("first child NodeName = %q, want #comment", first.NodeName())
	}
	if first.TextContent() != " a comment " {
		t.Errorf("comment TextContent = %q, want %q", first.TextContent(), " a comment ")
	}
}
