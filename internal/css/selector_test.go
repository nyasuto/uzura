package css

import (
	"strings"
	"testing"

	"github.com/nyasuto/uzura/internal/dom"
	htmlparser "github.com/nyasuto/uzura/internal/html"
)

func parseHTML(t *testing.T, s string) *dom.Document {
	t.Helper()
	doc, err := htmlparser.Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return doc
}

func TestCompile(t *testing.T) {
	tests := []struct {
		name    string
		sel     string
		wantErr bool
	}{
		{"tag", "div", false},
		{"class", ".foo", false},
		{"id", "#bar", false},
		{"attribute", "[href]", false},
		{"complex", "div.foo > span.bar", false},
		{"invalid", "!!!", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.sel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compile(%q) error = %v, wantErr %v", tt.sel, err, tt.wantErr)
			}
		})
	}
}

func TestQuerySelectorAll(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="a" class="foo">
			<span class="bar">hello</span>
			<span class="baz">world</span>
		</div>
		<div id="b" class="foo">
			<p>text</p>
		</div>
	</body></html>`)

	tests := []struct {
		name     string
		sel      string
		wantLen  int
		wantTags []string
	}{
		{"by tag", "span", 2, []string{"SPAN", "SPAN"}},
		{"by class", ".foo", 2, []string{"DIV", "DIV"}},
		{"by id", "#a", 1, []string{"DIV"}},
		{"descendant", "div span", 2, []string{"SPAN", "SPAN"}},
		{"child combinator", "div > p", 1, []string{"P"}},
		{"attribute", "[id]", 2, []string{"DIV", "DIV"}},
		{"compound", "div.foo", 2, []string{"DIV", "DIV"}},
		{"no match", ".nonexistent", 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := QuerySelectorAll(doc, tt.sel)
			if err != nil {
				t.Fatalf("QuerySelectorAll error: %v", err)
			}
			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
			for i, r := range results {
				if i < len(tt.wantTags) && r.TagName() != tt.wantTags[i] {
					t.Errorf("result[%d].TagName() = %q, want %q", i, r.TagName(), tt.wantTags[i])
				}
			}
		})
	}
}

func TestQuerySelector(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="first" class="item">one</div>
		<div id="second" class="item">two</div>
	</body></html>`)

	tests := []struct {
		name   string
		sel    string
		wantID string
	}{
		{"first match", ".item", "first"},
		{"by id", "#second", "second"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := QuerySelector(doc, tt.sel)
			if err != nil {
				t.Fatalf("QuerySelector error: %v", err)
			}
			if result == nil {
				t.Fatal("expected a result, got nil")
			}
			if result.Id() != tt.wantID {
				t.Errorf("got id=%q, want %q", result.Id(), tt.wantID)
			}
		})
	}

	// No match
	result, err := QuerySelector(doc, ".nonexistent")
	if err != nil {
		t.Fatalf("QuerySelector error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestQueryFromElement(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="container">
			<span class="inner">inside</span>
		</div>
		<span class="outer">outside</span>
	</body></html>`)

	container := doc.GetElementById("container")
	if container == nil {
		t.Fatal("container not found")
	}

	// QuerySelectorAll from element should only find descendants
	results, err := QuerySelectorAll(container, "span")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results from container, want 1", len(results))
	}
	if len(results) > 0 && results[0].ClassName() != "inner" {
		t.Errorf("got class=%q, want %q", results[0].ClassName(), "inner")
	}
}

func TestDocumentQuerySelector(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="a" class="item">one</div>
		<div id="b" class="item">two</div>
	</body></html>`)

	// Document.QuerySelector
	elem, err := doc.QuerySelector(".item")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if elem == nil || elem.Id() != "a" {
		t.Errorf("expected first .item (id=a), got %v", elem)
	}

	// Document.QuerySelectorAll
	elems, err := doc.QuerySelectorAll(".item")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(elems) != 2 {
		t.Errorf("expected 2 .item elements, got %d", len(elems))
	}
}

func TestElementQuerySelector(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="parent">
			<span class="child">inside</span>
		</div>
		<span class="sibling">outside</span>
	</body></html>`)

	parent := doc.GetElementById("parent")
	if parent == nil {
		t.Fatal("parent not found")
	}

	// Element.QuerySelector — only descendants
	elem, err := parent.QuerySelector("span")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if elem == nil || elem.ClassName() != "child" {
		t.Errorf("expected span.child, got %v", elem)
	}

	// Element.QuerySelectorAll — only descendants
	elems, err := parent.QuerySelectorAll("span")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(elems) != 1 {
		t.Errorf("expected 1 span inside parent, got %d", len(elems))
	}
}

func TestMatches(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="target" class="foo bar">text</div>
	</body></html>`)

	target := doc.GetElementById("target")
	if target == nil {
		t.Fatal("target not found")
	}

	tests := []struct {
		name string
		sel  string
		want bool
	}{
		{"class match", ".foo", true},
		{"multi class", ".foo.bar", true},
		{"id match", "#target", true},
		{"tag match", "div", true},
		{"no match", ".baz", false},
		{"wrong tag", "span", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Matches(target, tt.sel)
			if err != nil {
				t.Fatalf("Matches error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.sel, got, tt.want)
			}
		})
	}

	// Also test via Element.Matches
	got, err := target.Matches(".foo")
	if err != nil {
		t.Fatalf("Element.Matches error: %v", err)
	}
	if !got {
		t.Error("Element.Matches('.foo') should be true")
	}
}

func TestClosest(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div class="outer">
			<div class="inner">
				<span id="deep">text</span>
			</div>
		</div>
	</body></html>`)

	deep := doc.GetElementById("deep")
	if deep == nil {
		t.Fatal("deep not found")
	}

	tests := []struct {
		name      string
		sel       string
		wantClass string
		wantNil   bool
	}{
		{"self", "span", "", false},
		{"parent", ".inner", "inner", false},
		{"ancestor", ".outer", "outer", false},
		{"body", "body", "", false},
		{"no match", ".nonexistent", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Closest(deep, tt.sel)
			if err != nil {
				t.Fatalf("Closest error: %v", err)
			}
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected a result, got nil")
			}
			if tt.wantClass != "" && result.ClassName() != tt.wantClass {
				t.Errorf("got class=%q, want %q", result.ClassName(), tt.wantClass)
			}
		})
	}

	// Also test via Element.Closest
	result, err := deep.Closest(".outer")
	if err != nil {
		t.Fatalf("Element.Closest error: %v", err)
	}
	if result == nil || result.ClassName() != "outer" {
		t.Errorf("Element.Closest('.outer') unexpected result")
	}
}

func TestComplexSelectors(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<div id="nav" class="main-nav">
			<ul>
				<li class="active"><a href="/home">Home</a></li>
				<li><a href="/about" class="link external">About</a></li>
				<li><a href="/contact">Contact</a></li>
			</ul>
		</div>
		<div id="content">
			<article class="post featured">
				<h2>Title</h2>
				<p class="summary">Summary text</p>
			</article>
			<article class="post">
				<h2>Other</h2>
			</article>
		</div>
	</body></html>`)

	tests := []struct {
		name    string
		sel     string
		wantLen int
	}{
		{"descendant + attribute", "div a[href]", 3},
		{"child + class", "ul > li.active", 1},
		{"sibling combinator", "article.featured ~ article", 1},
		{"multi-class compound", "a.link.external", 1},
		{"nested descendant", "#nav ul li a", 3},
		{"attribute value", `a[href="/about"]`, 1},
		{"attribute prefix", `a[href^="/"]`, 3},
		{"not pseudo", "li:not(.active)", 2},
		{"universal in context", "#content *", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := QuerySelectorAll(doc, tt.sel)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestPseudoClassSelectors(t *testing.T) {
	doc := parseHTML(t, `<html><body>
		<ul>
			<li class="a">first</li>
			<li class="b">second</li>
			<li class="c">third</li>
		</ul>
	</body></html>`)

	tests := []struct {
		name      string
		sel       string
		wantClass string
	}{
		{"first-child", "li:first-child", "a"},
		{"last-child", "li:last-child", "c"},
		{"nth-child(2)", "li:nth-child(2)", "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem, err := doc.QuerySelector(tt.sel)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if elem == nil {
				t.Fatal("expected a result, got nil")
			}
			if elem.ClassName() != tt.wantClass {
				t.Errorf("got class=%q, want %q", elem.ClassName(), tt.wantClass)
			}
		})
	}
}
